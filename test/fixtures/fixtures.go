// Created at 2026-06-28
package fixtures

import "time"

// User — структура с простым ключом
type User struct {
	ID    int64 `qqm:"pk"`
	Name  string
	Email string
}

// UserWithAge — структура с простым ключом и возрастом
type UserWithAge struct {
	ID    int64 `qqm:"pk"`
	Name  string
	Email string
	Age   int
}

// OrgUser — структура с составным ключом
type OrgUser struct {
	OrgID  int64 `qqm:"pk"`
	UserID int64 `qqm:"pk"`
	Name   string
	Email  string
}

func (o *OrgUser) SQLName() string { return "org_users" }

// RoomID — простой ключ с автогенерацией
type RoomID struct {
	ID int64 `qqm:"pk;auto"`
}

// MappingRoomID — ключ комнаты для составного ключа (без auto)
type MappingRoomID struct {
	ID int64 `qqm:"pk"`
}

// TeacherID — тип для составного ключа
type TeacherID int64

// TeacherKey — ключ преподавателя
type TeacherKey struct {
	Key TeacherID `qqm:"pk;col=ID"`
}

// Rooms — таблица комнат
type Rooms struct {
	RoomID
	Name      string
	Square    float64
	CreatedAt int64 `qqm:"auto"`
}

// RoomMapping — таблица связей комнат и преподавателей
type RoomMapping struct {
	MappingRoomID `qqm:"prefix=room_;ref=rooms.id"`
	TeacherKey    `qqm:"prefix=teacher_"`
	From          int64 `qqm:"col=time_from"`
	To            int64 `qqm:"col=time_to"`
	CreatedAt     int64 `qqm:"auto"`
}

// FullRoomMapping — полная связь с автором
type FullRoomMapping struct {
	RoomMapping
	Author string `qqm:"col=author_name"`
}

// SomeID — тип для SomeTable
type SomeID int64

// SomeTable — структура с anonymous non-struct PK и auto полем
type SomeTable struct {
	SomeID  `qqm:"pk"`
	FieldRW string
	FieldRO time.Time `qqm:"auto"`
}

// EmbeddedPK — структура с PK для тестов anonymous struct
type EmbeddedPK struct {
	ID int64 `qqm:"pk;col=id"`
}

// EmbeddedFields — структура с полями для тестов prefix
type EmbeddedFields struct {
	Name  string `qqm:"col=name"`
	Email string `qqm:"col=email"`
}

// RowWithEmbeddedPK — структура с embedded PK и prefix
type RowWithEmbeddedPK struct {
	EmbeddedPK
	EmbeddedFields `qqm:"prefix=usr_"`
	Status         string `qqm:"col=status"`
}

// DeepNested — структура для тестов глубокой вложенности
type DeepNested struct {
	EmbeddedFields `qqm:"prefix=deep_"`
	Extra          int `qqm:"col=extra"`
}

// RowWithDeepEmbed — структура с глубокой вложенностью
type RowWithDeepEmbed struct {
	DeepNested `qqm:"prefix=nested_"`
	TopField   string `qqm:"col=top_field"`
}

// AutoEmbedded — структура с auto и readonly полями
type AutoEmbedded struct {
	CreatedAt string `qqm:"col=created_at;auto"`
	UpdatedAt string `qqm:"col=updated_at;readonly"`
}

// RowWithAutoEmbed — структура с embedded auto/readonly полями
type RowWithAutoEmbed struct {
	ID int64 `qqm:"pk"`
	AutoEmbedded
	Value string `qqm:"col=value"`
}

// PKWithAuto — структура с auto PK
type PKWithAuto struct {
	ID int64 `qqm:"pk;auto"`
}

// RowWithPKAuto — структура с embedded auto PK
type RowWithPKAuto struct {
	PKWithAuto
	Name string `qqm:"col=name"`
}
