# SUDO (Suck it Up and DO it)

A free and open-source kanban board built with Go, HTMX, and TailwindCSS for efficient team task management.

## Overview

SUDO is a modern kanban board application that helps teams organize, track, and complete tasks. Built with performance and simplicity in mind, it provides essential project management features without unnecessary complexity.

## Core Features

- **Authentication**: Email-based OTP login system
- **Multi-board Support**: Create unlimited project boards with custom columns
- **Task Management**: Full CRUD operations for tasks with drag-and-drop functionality
- **Team Collaboration**: Member invites and task assignments
- **Nested Boards**: Convert complex tasks into sub-boards
- **Global Search**: Search across all boards and tasks
- **Real-time Collaboration**: WebSocket-powered live updates
- **Responsive Design**: Mobile and desktop optimized interface

## Technical Stack

- **Backend**: Go 1.24+ with Gin web framework
- **Frontend**: HTMX for dynamic interactions, TailwindCSS for styling
- **Templates**: Templ for type-safe Go templating
- **Database**: PostgreSQL via Supabase
- **Authentication**: Session-based with secure OTP delivery
- **Real-time**: WebSocket connections for live updates

## Quick Start

1. Install dependencies:
   ```bash
   make install
   ```

2. Setup environment:
   ```bash
   make setup
   # Edit .env with your Supabase credentials
   ```

3. Run database migrations in your Supabase dashboard

4. Start development server:
   ```bash
   make dev
   ```

5. Access application at http://localhost:8080

## Development Commands

```bash
make install    # Install dependencies and tools
make dev        # Start development server with hot reload
make build      # Production build
make test       # Run test suite
make clean      # Clean generated files
make help       # Full command reference
```

## Architecture

### System Design

SUDO follows a layered architecture with clear separation of concerns:

**Presentation Layer**
- HTMX-powered frontend with server-side rendering
- Templ templates for type-safe HTML generation
- TailwindCSS for utility-first styling
- WebSocket connections for real-time updates

**Application Layer**
- Gin HTTP router handling requests
- Session-based authentication middleware
- Request/response handlers with business logic
- WebSocket handlers for real-time features

**Service Layer**
- Database service for data persistence
- Email service for OTP delivery
- Authentication service for user management
- Task and board management services

**Data Layer**
- PostgreSQL database via Supabase
- RESTful API integration
- Session storage for authentication state

### Key Architectural Decisions

**Server-Side Rendering with HTMX**
- Reduces JavaScript complexity
- Provides progressive enhancement
- Maintains fast page loads with dynamic updates

**Session-Based Authentication**
- Secure cookie-based sessions
- OTP verification via email
- Middleware-based route protection

**Real-time Collaboration**
- WebSocket connections per board
- Event-driven updates for task changes
- Optimistic UI updates with server reconciliation

**Static Asset Management**
- Development: Cache-busting headers for instant updates
- Production: Aggressive caching for performance
- TailwindCSS compilation with purging

### Request Flow

1. Client sends HTTP request to Gin router
2. Authentication middleware validates session
3. Handler processes business logic
4. Service layer interacts with database
5. Templ renders HTML response
6. HTMX updates DOM without full page reload

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        CLIENT BROWSER                           │
├─────────────────────────────────────────────────────────────────┤
│  HTML/CSS (TailwindCSS)  │  HTMX  │  WebSocket  │  Static Assets│
└─────────────────────────────────────────────────────────────────┘
                                    │
                                    │ HTTP/WS
                                    ▼
┌─────────────────────────────────────────────────────────────────┐
│                      GIN WEB SERVER                             │
├─────────────────────────────────────────────────────────────────┤
│                     Middleware Stack                            │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────────────────┐ │
│  │   Sessions   │ │     Auth     │ │    Static/Cache Control  │ │
│  │   Handler    │ │  Middleware  │ │       Middleware         │ │
│  └──────────────┘ └──────────────┘ └──────────────────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│                        Route Handlers                           │
│  ┌────────────┐ ┌────────────┐ ┌──────────┐ ┌─────────────────┐ │
│  │    Auth    │ │   Boards   │ │  Tasks   │ │   WebSocket     │ │
│  │  Handlers  │ │  Handlers  │ │ Handlers │ │    Handlers     │ │
│  └────────────┘ └────────────┘ └──────────┘ └─────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                                    │
                                    │ Service Calls
                                    ▼
┌─────────────────────────────────────────────────────────────────┐
│                       SERVICE LAYER                             │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────────────────┐ │
│  │   Database   │ │    Email     │ │      Template            │ │
│  │   Service    │ │   Service    │ │     Rendering            │ │
│  │              │ │   (OTP)      │ │      (Templ)             │ │
│  └──────────────┘ └──────────────┘ └──────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                                    │
                                    │ Data Access
                                    ▼
┌─────────────────────────────────────────────────────────────────┐
│                    SUPABASE (PostgreSQL)                        │
├─────────────────────────────────────────────────────────────────┤
│  ┌───────────┐ ┌──────────┐ ┌─────────┐ ┌─────────────────────┐ │
│  │   Users   │ │  Boards  │ │ Columns │ │       Tasks         │ │
│  │           │ │          │ │         │ │                     │ │
│  │ - id      │ │ - id     │ │ - id    │ │ - id                │ │
│  │ - email   │ │ - name   │ │ - name  │ │ - title             │ │
│  │ - created │ │ - owner  │ │ - order │ │ - description       │ │
│  └───────────┘ └──────────┘ └─────────┘ │ - assigned_to       │ │
│                                         │ - column_id         │ │
│  ┌─────────────────────────────────────┐│ - board_id          │ │
│  │        BoardMembers                 ││ - nested_board_id   │ │
│  │                                     │└─────────────────────┘ │
│  │ - board_id                          │                        │
│  │ - user_id                           │                        │
│  │ - role                              │                        │
│  └─────────────────────────────────────┘                        │
└─────────────────────────────────────────────────────────────────┘
```

### Database Schema

**Core Entities:**
- **Users**: Authentication and user management
- **Boards**: Project containers with columns and members
- **Columns**: Task organization within boards (To Do, In Progress, Done)
- **Tasks**: Work items with assignments and metadata
- **BoardMembers**: Many-to-many relationship between users and boards

**Relationships:**
- Users → Boards (1:many, ownership)
- Boards → Columns (1:many)
- Boards → BoardMembers → Users (many:many)
- Columns → Tasks (1:many)
- Users → Tasks (1:many, assignments)
- Tasks → Boards (1:1, nested boards)

### Data Flow Examples

**Board Creation:**
Client → HTMX Request → Auth Middleware → Board Handler → Database Service → Supabase → Response → Templ Render → HTMX DOM Update

**Real-time Task Update:**
Task Move → WebSocket Handler → Database Update → Broadcast to Board Members → Live DOM Updates

**Authentication:**
Email Input → OTP Request → Email Service → Verify OTP → Session Creation → Protected Route Access

Detailed architectural patterns and debugging guides are documented in the learning directory.

## Learning Resources

- [Event Bubbling & Auth Middleware](learning/event-bubbling-and-auth-middleware.md) - Authentication patterns and event handling
- [Cache Busting & Static Assets](learning/cache-busting-and-static-assets.md) - Static asset management and caching strategies

## Project Structure

```
cmd/server/          # Application entry point
internal/
  handlers/          # HTTP request handlers
  database/          # Database connection and queries
  models/           # Data models
  email/            # Email service
templates/          # Templ templates
static/             # CSS, JS, and static assets
learning/           # Technical documentation
```

## Contributing

This is an open-source project. Contributions are welcome through pull requests and issue reports.
