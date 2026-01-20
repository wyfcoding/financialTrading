package application

type LoginCommand struct {
	Username string
	Password string
}

type AuthTokenDTO struct {
	Token     string
	Type      string
	ExpiresAt int64
}

type CreateAdminCommand struct {
	Username string
	Password string
	RoleID   uint
}

type AdminDTO struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	RoleName  string `json:"role_name"`
	CreatedAt int64  `json:"created_at"`
}
