package state

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"sync"
    "fmt"
)

type StateManager struct {
	filePath string
	mutex    sync.Mutex
	state    map[string]map[string]string
}

// NewStateManager создает экземпляр StateManager и загружает существующее состояние из файла, если он существует.
func NewStateManager() (*StateManager, error) {
    filePath := "state.json"

    manager := &StateManager{
		filePath: filePath,
		state:    make(map[string]map[string]string),
	}

    if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
        // Создаем новый пустой файл
        emptyState := make(map[string]map[string]string)
        data, err := json.MarshalIndent(emptyState, "", "  ")
        if err != nil {
            return nil, err
        }

        if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
            return nil, err
        }

        return manager, nil // Возвращаем менеджер с пустым состоянием
    }

	// Загрузка состояния из файла
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &manager.state)
	if err != nil {
		return nil, err
	}

	return manager, nil
}


func (sm *StateManager) AddOption(unitNodeUUID, optionName, optionValue string) error {
    sm.mutex.Lock()
    fmt.Println("Locked mutex in AddOption")

    if _, exists := sm.state[unitNodeUUID]; !exists {
        sm.state[unitNodeUUID] = make(map[string]string)
    }

    sm.state[unitNodeUUID][optionName] = optionValue
    fmt.Println("State updated, calling Save")

    sm.mutex.Unlock()

    return sm.Save()
}

func (sm *StateManager) Save() error {
    fmt.Println("Starting Save")
    sm.mutex.Lock()
    fmt.Println("Locked mutex in Save")
    defer func() {
        fmt.Println("Unlocking mutex in Save")
        sm.mutex.Unlock()
    }()

    data, err := json.MarshalIndent(sm.state, "", "  ")
    if err != nil {
        fmt.Println("Error marshaling state:", err)
        return err
    }

    err = ioutil.WriteFile(sm.filePath, data, 0644)
    if err != nil {
        fmt.Println("Error writing to file:", err)
        return err
    }

    fmt.Println("State saved successfully")
    return nil
}


// RemoveOption удаляет указанную опцию из UnitNode с указанным UUID.
func (sm *StateManager) RemoveOption(unitNodeUUID, optionName string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if _, exists := sm.state[unitNodeUUID]; !exists {
		return errors.New("unit node UUID not found")
	}

	if _, exists := sm.state[unitNodeUUID][optionName]; !exists {
		return errors.New("option name not found")
	}

	delete(sm.state[unitNodeUUID], optionName)

	return sm.Save()
}

// GetState возвращает текущее состояние для всех UnitNode.
func (sm *StateManager) GetState() map[string]map[string]string {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Копируем состояние для безопасности
	copyState := make(map[string]map[string]string)
	for uuid, options := range sm.state {
		copyOptions := make(map[string]string)
		for k, v := range options {
			copyOptions[k] = v
		}
		copyState[uuid] = copyOptions
	}

	return copyState
}

