package fixtures

// V2User — структура пользователя с v2-тегами tbl.
type V2User struct {
	ID    int64  `tbl:"pk;auto"`
	Name  string
	Email string
}

func (V2User) SQLName() string { return "users" }

// V2Order — структура заказа с FK на V2User.
type V2Order struct {
	ID     int64   `tbl:"pk;auto"`
	UserID int64   `tbl:"ref=users.id"`
	Amount float64
}

func (V2Order) SQLName() string { return "orders" }

// V2UserWithOrder — QROW для INNER JOIN User + Order.
type V2UserWithOrder struct {
	User  V2User `tbl:"from"`
	Order V2Order
}

// V2UserWithOrderLeft — QROW для LEFT JOIN User + Order.
type V2UserWithOrderLeft struct {
	User  V2User  `tbl:"from"`
	Order V2Order `tbl:"join=left"`
}

// V2UserWithSort — структура пользователя с sort-тегами для тестов Many.
type V2UserWithSort struct {
	ID    int64  `tbl:"pk;auto"`
	Name  string `tbl:"sort=1"`
	Email string `tbl:"sort=2,desc"`
	Age   int
}

func (V2UserWithSort) SQLName() string { return "users" }

// V2UserWithSortAndOrder — QROW с сортировкой для тестов Query Many.
type V2UserWithSortAndOrder struct {
	User  V2UserWithSort `tbl:"from"`
	Order V2Order
}

// V2UserNoPK — структура пользователя без PK.
type V2UserNoPK struct {
	Name  string
	Email string
}

func (V2UserNoPK) SQLName() string { return "users" }
