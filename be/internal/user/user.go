package user

type User struct {
	ID       int    `json:"id"`
	Realname string `json:"realname"`
	Username string `json:"username"`
	Password string `json:"password"`
	Role     Role   `json:"role"`
}
