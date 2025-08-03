# SUDO (Suck it Up and DO it) - a free and open source kanban board

## Project Tree

```
kanban-app/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── database/
│   │   ├── migrations/
│   │   └── db.go
│   ├── handlers/
│   │   ├── auth.go
│   │   ├── board.go
│   │   ├── card.go
│   │   └── websocket.go
│   ├── middleware/
│   │   ├── auth.go
│   │   └── cors.go
│   ├── models/
│   │   ├── user.go
│   │   ├── board.go
│   │   └── card.go
│   ├── services/
│   │   ├── auth.go
│   │   ├── board.go
│   │   └── websocket.go
│   └── templates/
│       ├── layouts/
│       ├── components/
│       └── pages/
├── web/
│   ├── static/
│   │   ├── css/
│   │   └── js/
│   └── assets/
├── go.mod
├── go.sum
├── tailwind.config.js
└── README.md
```
