package application

type AuthTokenDTO struct {
	Token     string
	Type      string
	ExpiresAt int64
}

type AdminDTO struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	RoleName  string `json:"role_name"`
	CreatedAt int64  `json:"created_at"`
}

type RoleDTO struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Permissions string `json:"permissions"`
	CreatedAt   int64  `json:"created_at"`
}
