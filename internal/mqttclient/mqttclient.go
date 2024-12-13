package mqttclient

import (
	"fmt"
    "encoding/json"
    "runtime"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
	"math/rand"
	"picker/internal/config"
	"picker/internal/queries"
	"picker/internal/schema"
	"strings"
	"time"
)

// MqttClient структура для управления MQTT клиентом
type MqttClient struct {
	client mqtt.Client
}

type UnitState struct {
	Millis         int64   `json:"millis"`
	MemFree        uint64  `json:"mem_free"`
	MemAlloc       uint64  `json:"mem_alloc"`
	Freq           float64 `json:"freq"`
	CommitVersion  string  `json:"commit_version"`
}

func getUnitState() UnitState {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

    cfg := config.GetConfig()

	return UnitState{
		Millis:        time.Now().UnixNano() / int64(time.Millisecond),
		MemFree:       memStats.Frees,
		MemAlloc:      memStats.Alloc,
		Freq:          float64(runtime.NumCPU()), // Замените на реальную частоту CPU, если доступна
		CommitVersion: cfg.COMMIT_VERSION,
	}
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateRandomString generates a random string of specified length.
func GenerateRandomString(n int) string {
	rand.Seed(time.Now().UnixNano())
	sb := strings.Builder{}
	sb.Grow(n)
	for i := 0; i < n; i++ {
		sb.WriteByte(letterBytes[rand.Intn(len(letterBytes))])
	}
	return sb.String()
}

// New создаёт новый экземпляр MqttClient
func New() (*MqttClient, error) {
	cfg := config.GetConfig()

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", cfg.MQTT_URL, 1883))
	opts.SetClientID(cfg.UnitUUID)
	opts.SetUsername(cfg.PEPEUNIT_TOKEN)
	opts.SetPassword("putblic")
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(time.Duration(cfg.PING_INTERVAL) * time.Second)
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(10 * time.Second)

	client := mqtt.NewClient(opts)
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	fmt.Printf("Статус подключения после публикации: %s\n", client.IsConnected())
	fmt.Printf("Статус подключения после публикации: %s\n", client.IsConnectionOpen())

	return &MqttClient{client: client}, nil
}

// Publish отправляет сообщение в заданный топик
func (m *MqttClient) Publish(topic string, qos byte, retained bool, payload interface{}) error {
	token := m.client.Publish(topic, qos, retained, payload)
	token.Wait()
	return token.Error()
}

// publishState циклически публикует состояние
func publishState(client *MqttClient, topic string, interval time.Duration) {
	go func() {
		for {
			state := getUnitState()
			payload, err := json.Marshal(state)
			if err != nil {
				log.Printf("Ошибка сериализации состояния: %v", err)
				continue
			}
			err = client.Publish(topic, 0, false, payload)
			if err != nil {
				log.Printf("Ошибка публикации состояния: %v", err)
			}
            fmt.Println(state)
			time.Sleep(interval * time.Second)
		}
	}()
}

// Subscribe подписывается на заданный топик и обрабатывает входящие сообщения
func (m *MqttClient) Subscribe(topics []string, qos byte, callback func(client mqtt.Client, msg mqtt.Message)) error {
	filters := make(map[string]byte)
	for _, topic := range topics {
		filters[topic] = qos
	}

	token := m.client.SubscribeMultiple(filters, callback)
	token.Wait()
	return token.Error()
}

// Disconnect отключает клиента
func (m *MqttClient) Disconnect(quiesce uint) {
	m.client.Disconnect(quiesce)
}

type UpdateMessage struct {
	NewCommitVersion      string `json:"NEW_COMMIT_VERSION"`
	CompiledFirmwareLink string `json:"COMPILED_FIRMWARE_LINK"`
}

func RunMqttClient() (*MqttClient, error) {
	// Создание MQTT клиента
	client, err := New()
	if err != nil {
		log.Fatalf("Ошибка подключения к MQTT брокеру: %v", err)
	}

	// Обработчик входящих сообщений
	schemaUpdateHandler := func(client mqtt.Client, msg mqtt.Message) {

        var update UpdateMessage

        err := json.Unmarshal([]byte(msg.Payload()), &update)

        fmt.Println(update, err)

		fmt.Printf("Получено сообщение из топика %s: %s\n", msg.Topic(), msg.Payload())
	}
   
	// Обработчик входящих сообщений
	updateHandler := func(client mqtt.Client, msg mqtt.Message) {
		fmt.Printf("Получено сообщение из топика %s: %s\n", msg.Topic(), msg.Payload())
		newSchema, err := queries.GetCurrentSchema()
		if err == nil {
            err = schema.SaveSchema(newSchema)
		}
	}


	schemaData, err := schema.LoadSchema()
	if err == nil {
		// Подписка на топик
		err = client.Subscribe(schemaData.InputBaseTopic["schema_update/pepeunit"], 0, schemaUpdateHandler)
		if err != nil {
			log.Fatalf("Ошибка подписки на топик: %v", err)
		}

        err = client.Subscribe(schemaData.InputBaseTopic["update/pepeunit"], 0, updateHandler)
		if err != nil {
			log.Fatalf("Ошибка подписки на топик: %v", err)
		}
        
	}

    // Запуск циклической отправки состояния
	cfg := config.GetConfig()
	publishState(client, schemaData.OutputBaseTopic["state/pepeunit"][0], time.Duration(cfg.STATE_SEND_INTERVAL))

	return client, nil
}

// IsConnected проверяет, подключен ли клиент
func (m *MqttClient) IsConnected() bool {
	return m.client.IsConnected()
}

func (m *MqttClient) Connect() mqtt.Token {
	return m.client.Connect()
}
