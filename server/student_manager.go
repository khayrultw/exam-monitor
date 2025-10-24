package main

import (
	"image"
	"sort"
	"strings"
	"sync"
	"time"
)

type StudentManager struct {
	students       map[string]*Student
	sortedStudents []*Student
	sortField      string
	sortAsc        bool
	needsResort    bool
	lastSortTime   time.Time
	mu             sync.Mutex
}

func NewStudentManager() *StudentManager {
	return &StudentManager{
		students:       make(map[string]*Student),
		sortedStudents: make([]*Student, 0),
		sortField:      "name",
		sortAsc:        true,
		needsResort:    true,
	}
}

func (sm *StudentManager) Add(id, name string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	student := NewStudent(id, name)
	sm.students[id] = student
	sm.needsResort = true
}

func (sm *StudentManager) Remove(id string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.students, id)
	sm.needsResort = true
}

func (sm *StudentManager) Exists(id string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	_, ok := sm.students[id]
	return ok
}

func (sm *StudentManager) UpdateImage(id string, img image.Image) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	student, ok := sm.students[id]
	if !ok {
		return
	}
	student.UpdateImage(img)
}

func (sm *StudentManager) UpdateName(id, name string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	student, ok := sm.students[id]
	if !ok {
		return
	}
	if student.Name != name {
		student.Name = name
		sm.needsResort = true
	}
}

func (sm *StudentManager) GetSorted() []*Student {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.needsResort || time.Since(sm.lastSortTime) > 500*time.Millisecond {
		sm.sortedStudents = make([]*Student, 0, len(sm.students))
		for _, student := range sm.students {
			sm.sortedStudents = append(sm.sortedStudents, student)
		}

		sort.SliceStable(sm.sortedStudents, func(i, j int) bool {
			var result bool
			if sm.sortField == "name" {
				result = strings.ToLower(sm.sortedStudents[i].Name) < strings.ToLower(sm.sortedStudents[j].Name)
			} else {
				result = sm.sortedStudents[i].Id < sm.sortedStudents[j].Id
			}
			if !sm.sortAsc {
				result = !result
			}
			return result
		})

		sm.needsResort = false
		sm.lastSortTime = time.Now()
	}

	return sm.sortedStudents
}

func (sm *StudentManager) SetSortField(field string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.sortField = field
	sm.needsResort = true
}

func (sm *StudentManager) ToggleSortDirection() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.sortAsc = !sm.sortAsc
	sm.needsResort = true
}

func (sm *StudentManager) GetSortField() string {
	return sm.sortField
}

func (sm *StudentManager) IsSortAscending() bool {
	return sm.sortAsc
}

func (sm *StudentManager) Clear() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.students = make(map[string]*Student)
	sm.sortedStudents = make([]*Student, 0)
	sm.needsResort = true
}

func (sm *StudentManager) Count() int {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	return len(sm.students)
}

func (sm *StudentManager) GetByID(id string) *Student {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	return sm.students[id]
}
