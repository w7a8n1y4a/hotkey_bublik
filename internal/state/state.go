package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"picker/internal/queries"
    "strconv"
)

type StateManager struct {
	mutex sync.Mutex
	state map[string][][]string
}

// NewStateManager создает экземпляр StateManager и загружает существующее состояние из удаленного хранилища, если оно существует.
func NewStateManager() (*StateManager, error) {
	manager := &StateManager{
		state: make(map[string][][]string),
	}

	// Получаем состояние из удаленного хранилища
	serializedState, err := queries.GetStateStorage()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve state from storage: %w", err)
	}
    
	if serializedState == "\"\"" {
		// Если состояние пустое, возвращаем менеджер с пустым состоянием
		return manager, nil
	}
    
    s, _ := strconv.Unquote(string(serializedState))

	// Десериализуем состояние
	err = json.Unmarshal([]byte(s), &manager.state)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return manager, nil
}

// AddOption добавляет новую пару ["optionName", "optionValue"] в массив по unitNodeUUID.
func (sm *StateManager) AddOption(unitNodeUUID, optionName, optionValue string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if _, exists := sm.state[unitNodeUUID]; !exists {
		sm.state[unitNodeUUID] = [][]string{}
	}

	// Проверяем, есть ли уже такая опция, и обновляем её, если найдена
	for i, pair := range sm.state[unitNodeUUID] {
		if pair[0] == optionName {
			sm.state[unitNodeUUID][i][1] = optionValue
			return sm.Save()
		}
	}

	// Добавляем новую опцию, если её не было
	sm.state[unitNodeUUID] = append(sm.state[unitNodeUUID], []string{optionName, optionValue})

	return sm.Save()
}

// RemoveOption удаляет указанную опцию из UnitNode с указанным UUID.
func (sm *StateManager) RemoveOption(unitNodeUUID, optionName string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if _, exists := sm.state[unitNodeUUID]; !exists {
		return errors.New("unit node UUID not found")
	}

	newOptions := [][]string{}
	found := false

	// Фильтруем опции, исключая удаляемую
	for _, pair := range sm.state[unitNodeUUID] {
		if pair[0] == optionName {
			found = true
			continue
		}
		newOptions = append(newOptions, pair)
	}

	if !found {
		return errors.New("option name not found")
	}

	sm.state[unitNodeUUID] = newOptions

	return sm.Save()
}

// Save сохраняет текущее состояние в удаленное хранилище.
func (sm *StateManager) Save() error {
	data, err := json.Marshal(sm.state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	err = queries.SetStateStorage(string(data))
	if err != nil {
		return fmt.Errorf("failed to save state to storage: %w", err)
	}

	fmt.Println("State saved successfully")
	return nil
}

// GetState возвращает копию текущего состояния для всех UnitNode.
func (sm *StateManager) GetState() map[string][][]string {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Копируем состояние для безопасности
	copyState := make(map[string][][]string)
	for uuid, options := range sm.state {
		copyOptions := make([][]string, len(options))
		for i, pair := range options {
			copyOptions[i] = append([]string{}, pair...)
		}
		copyState[uuid] = copyOptions
	}

	return copyState
}

