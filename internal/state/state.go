package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
)

type StateManager struct {
	filePath string
	mutex    sync.Mutex
	state    map[string][][]string
}

// NewStateManager создает экземпляр StateManager и загружает существующее состояние из файла, если он существует.
func NewStateManager() (*StateManager, error) {
	filePath := "state.json"

	manager := &StateManager{
		filePath: filePath,
		state:    make(map[string][][]string),
	}

	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		// Создаем новый пустой файл
		emptyState := make(map[string][][]string)
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
    
    sm.mutex.Unlock()

    fmt.Println("Success remove option")

	return sm.Save()
}

// Save сохраняет текущее состояние в файл.
func (sm *StateManager) Save() error {
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

