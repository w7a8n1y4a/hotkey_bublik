package mqttclient

import (
	"fmt"

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

func RunMqttClient() (*MqttClient, error) {
	// Создание MQTT клиента
	client, err := New()
	if err != nil {
		log.Fatalf("Ошибка подключения к MQTT брокеру: %v", err)
	}

	// Обработчик входящих сообщений
	messageHandler := func(client mqtt.Client, msg mqtt.Message) {
		fmt.Printf("Получено сообщение из топика %s: %s\n", msg.Topic(), msg.Payload())
		newSchema, err := queries.GetCurrentSchema()
		if err == nil {
            err = schema.SaveSchema(newSchema)
		}
	}

	schemaData, err := schema.LoadSchema()
	if err == nil {
		// Подписка на топик
		err = client.Subscribe(schemaData.InputBaseTopic["schema_update/pepeunit"], 0, messageHandler)
		if err != nil {
			log.Fatalf("Ошибка подписки на топик: %v", err)
		}
	}

	return client, nil
}

// IsConnected проверяет, подключен ли клиент
func (m *MqttClient) IsConnected() bool {
	return m.client.IsConnected()
}

func (m *MqttClient) Connect() mqtt.Token {
	return m.client.Connect()
}
