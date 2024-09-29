package models

type Todo struct {
	Id int `db:"id"`
	Name string `db:"name"`
	Completed bool `db:"completed"`
	CompletedAt *string `db:"completedAt"`
	CreatedAt string `db:"createdAt"`
	UpdatedAt string `db:"updatedAt"`
	DeletedAt *string `db:"deletedAt"`
}
