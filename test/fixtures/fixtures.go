//nolint:goconst
package fixtures

import "time"

// User — структура с простым ключом
type User struct {
	ID    int64 `tbl:"pk;auto"`
	Name  string
	Email string
}

func (u *User) SQLName() string { return "users" }

// UserWithAge — структура с простым ключом и возрастом
type UserWithAge struct {
	ID    int64 `tbl:"pk;auto"`
	Name  string
	Email string
	Age   int
}

// OrgUser — структура с составным ключом
type OrgUser struct {
	OrgID  int64 `tbl:"pk"`
	UserID int64 `tbl:"pk"`
	Name   string
	Email  string
}

func (o *OrgUser) SQLName() string { return "org_users" }

// RoomID — простой ключ с автогенерацией
type RoomID struct {
	ID int64 `tbl:"pk;auto"`
}

// MappingRoomID — ключ комнаты для составного ключа (без auto)
type MappingRoomID struct {
	ID int64 `tbl:"pk"`
}

// TeacherID — тип для составного ключа
type TeacherID int64

// TeacherKey — ключ преподавателя
type TeacherKey struct {
	Key TeacherID `tbl:"pk;col=ID"`
}

// Rooms — таблица комнат
type Rooms struct {
	RoomID
	Name      string
	Square    float64
	CreatedAt int64 `tbl:"auto;upd"`
}

// RoomMapping — таблица связей комнат и преподавателей
type RoomMapping struct {
	MappingRoomID `tbl:"prefix=room_;ref=rooms.id"`
	TeacherKey    `tbl:"prefix=teacher_"`
	From          int64 `tbl:"col=time_from"`
	To            int64 `tbl:"col=time_to"`
	CreatedAt     int64 `tbl:"auto"`
}

// FullRoomMapping — полная связь с автором
type FullRoomMapping struct {
	RoomMapping
	Author string `tbl:"col=author_name"`
}

// SomeID — тип для SomeTable
type SomeID int64

// SomeTable — структура с anonymous non-struct PK и auto полем
type SomeTable struct {
	SomeID  `tbl:"pk;auto"`
	FieldRW string
	FieldRO time.Time `tbl:"auto"`
}

// EmbeddedPK — структура с PK для тестов anonymous struct
type EmbeddedPK struct {
	ID int64 `tbl:"pk;col=id"`
}

// EmbeddedFields — структура с полями для тестов prefix
type EmbeddedFields struct {
	Name  string `tbl:"col=name"`
	Email string `tbl:"col=email"`
}

// RowWithEmbeddedPK — структура с embedded PK и prefix
type RowWithEmbeddedPK struct {
	EmbeddedPK
	EmbeddedFields `tbl:"prefix=usr_"`
	Status         string `tbl:"col=status"`
}

// DeepNested — структура для тестов глубокой вложенности
type DeepNested struct {
	EmbeddedFields `tbl:"prefix=deep_"`
	Extra          int `tbl:"col=extra"`
}

// RowWithDeepEmbed — структура с глубокой вложенностью
type RowWithDeepEmbed struct {
	DeepNested `tbl:"prefix=nested_"`
	TopField   string `tbl:"col=top_field"`
}

// AutoEmbedded — структура с auto и update полями
type AutoEmbedded struct {
	CreatedAt string `tbl:"col=created_at;auto"`
	UpdatedAt string `tbl:"col=updated_at;auto;upd"`
}

// RowWithAutoEmbed — структура с embedded auto/update полями
type RowWithAutoEmbed struct {
	ID int64 `tbl:"pk"`
	AutoEmbedded
	Value string `tbl:"col=value"`
}

// PKWithAuto — структура с auto PK
type PKWithAuto struct {
	ID int64 `tbl:"pk;auto"`
}

// RowWithPKAuto — структура с embedded auto PK
type RowWithPKAuto struct {
	PKWithAuto
	Name string `tbl:"col=name"`
}

// Address — структура для тестирования префикса на именованных полях-структурах
type Address struct {
	City   string
	Street string
	Zip    string
}

// PersonWithAddress — структура с двумя именованными полями-структурами с префиксами
type PersonWithAddress struct {
	ID          int64 `tbl:"pk"`
	Name        string
	HomeAddress Address `tbl:"prefix=home_"`
	WorkAddress Address `tbl:"prefix=work_"`
}

// Order — структура заказа с FK на User
type Order struct {
	ID     int64 `tbl:"pk;auto"`
	UserID int64 `tbl:"ref=users.id"`
	Amount float64
}

func (o *Order) SQLName() string { return "orders" }

// OrderItem — структура позиции заказа с FK на Order
type OrderItem struct {
	ID       int64 `tbl:"pk;auto"`
	OrderID  int64 `tbl:"ref=orders.id"`
	Quantity int
	Price    float64
}

func (oi *OrderItem) SQLName() string { return "order_items" }

// UserWithOrder — Query-структура: User + Order (INNER JOIN)
type UserWithOrder struct {
	User  User `tbl:"from"`
	Order Order
}

// UserWithOrderLeft — Query-структура: User + Order (LEFT JOIN)
type UserWithOrderLeft struct {
	User  User  `tbl:"from"`
	Order Order `tbl:"join=left"`
}

// UserOrderItem — Query-структура с тремя таблицами: User + Order + OrderItem
type UserOrderItem struct {
	User      User `tbl:"from"`
	Order     Order
	OrderItem *OrderItem
}

// UserWithSort — структура с sort-тегами
type UserWithSort struct {
	ID    int64  `tbl:"pk;auto"`
	Name  string `tbl:"sort=1"`
	Email string `tbl:"sort=2,desc"`
	Age   int
}

func (u *UserWithSort) SQLName() string { return "users" }

// UserWithSortMulti — структура с несколькими sort-полями и DESC
type UserWithSortMulti struct {
	ID    int64  `tbl:"pk;auto"`
	Name  string `tbl:"sort=2"`
	Email string `tbl:"sort=1,desc"`
	Age   int    `tbl:"sort=3"`
}

func (u *UserWithSortMulti) SQLName() string { return "user_with_sort_multi" }

// UserWithSortAndOrder — QROW с сортировкой для тестов Query Many
type UserWithSortAndOrder struct {
	User  UserWithSort `tbl:"from"`
	Order Order
}

// UserNoPK — структура пользователя без PK
type UserNoPK struct {
	Name  string
	Email string
}

func (u *UserNoPK) SQLName() string { return "users" }

// OrderWithSort — структура заказа с sort для Query-тестов
type OrderWithSort struct {
	ID     int64   `tbl:"pk;auto"`
	UserID int64   `tbl:"ref=users.id;sort=1"`
	Amount float64 `tbl:"sort=2,desc"`
}

func (o *OrderWithSort) SQLName() string { return "orders" }
