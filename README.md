# SUDO (Suck it Up and DO )

A free, source-available kanban board built with Go, HTMX, and TailwindCSS for efficient team task management.

> 📖 **New to SUDO?** Check out our comprehensive guides:
> - [🚀 Self-Hosting Guide](/docs/SELF_HOST.md) - Deploy on your own infrastructure
> - [🔒 Security Documentation](/docs/SECURITY.md) - Encryption and security implementation
> - [🧪 Testing Guide](/docs/TESTING_SETUP_GUIDE.md) - Complete testing setup
> - [🔧 GitHub Workflow](/docs/GITHUB_WORKFLOW.md) - Development and deployment workflow

## Overview

SUDO is a kanban board application that helps people/teams organize, track, and complete tasks. Built with privacy, security and simplicity in mind, it provides essential project management features without unnecessary complexity. Self-host it for free or use it as-is for personal projects.

## Core Features

### 🔐 Authentication & Security
- **Email-based OTP login** - Passwordless authentication via one-time codes valid for 30 days at a time.
- **Military-grade encryption** - AES-256-GCM for data at rest
- **Session management** - Secure cookie-based sessions
- **Row-level security** - PostgreSQL RLS policies, access control at every step

### 📋 Board & Task Management
- **Multi-board support** - Create unlimited project boards with custom columns
- **Drag-and-drop interface** - Intuitive task movement between columns
- **Nested boards** - Convert complex tasks into sub-boards for better organization
- **Task completion tracking** - Mark tasks complete with visual indicators
- **Multiple assignees** - Assign tasks to multiple team members
- **Rich task details** - Titles, descriptions, deadlines, and priorities

### 👥 Collaboration
- **Team invitations** - Invite members via email to boards
- **Real-time updates** - WebSocket-powered live collaboration
- **User presence** - See who's online and working on the same board
- **Contact management** - Manage collaborators across all your boards
- **Board permissions** - Owner and member roles

### 🎨 User Experience
- **Dark/Light mode** - System-aware theme with manual toggle
- **Profile customization** - Upload avatars from URL or local files
- **Global search** - Search across all boards and tasks
- **Responsive design** - Mobile, tablet, and desktop optimized
- **Keyboard shortcuts** - Efficient navigation and actions

### ⚙️ Settings & Privacy
- **Profile management** - Update name, email, and avatar
- **Account deletion** - Complete data removal with confirmation
- **Invite management** - Add/remove collaborators from boards
- **DELETE ALL DATA ANYTIME** - just press the delete account button and nothing is retained even in my DB 

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

## Deployment

### Quick Deploy (Railway/Cloud)

The easiest way to deploy SUDO:

```bash
# 1. Fork this repository
# 2. Connect to Railway/Render/Fly.io
# 3. Add environment variables
# 4. Deploy!
```

See [Railway deployment docs](https://docs.railway.app/) for detailed steps.

### Self-Hosting

For full control and free hosting, see our **[Self-Hosting Guide](SELF_HOST.md)**:

- 🟢 **Simple Setup** (~10 min) - Perfect for personal use
- 🟡 **Production Setup** (~1 hour) - For teams with monitoring

**Deployment options:**
- [x] Docker (recommended)
- [x] Docker Compose with nginx
- [ ] Bare metal / VPS (Work in progress)

## Learning Resources

### 📚 Official Documentation

- **[Self-Hosting Guide](SELF_HOST.md)** - Complete deployment guide with architecture diagrams
- **[Security Documentation](SECURITY.md)** - Encryption, OTP hashing, and security best practices
- **[Testing Guide](TESTING_SETUP_GUIDE.md)** - Unit tests, integration tests, and load testing

## Project Structure

```
```

## Contributing

We welcome contributions! Here's how you can help:

### Ways to Contribute

I am fresh graduate, this code can only get better, open to any help/pointers/contributers:

- 🐛 **Report Bugs** - Open an issue with detailed reproduction steps
- 💡 **Suggest Features** - Share your ideas in GitHub Discussions
- 📝 **Improve Documentation** - Fix typos, add examples, or write guides
- 🔧 **Submit Code** - Fix bugs or implement new features
- 🎨 **Design Improvements** - UI/UX enhancements
- 🧪 **Write Tests** - Improve test coverage

### Development Setup

```bash
# 1. Fork and clone the repository
git clone https://github.com/yourusername/sudo.git
cd sudo

# 2. Install dependencies
npm install
go mod download

# 3. Set up environment
cp .env.example .env
# Edit .env with your Supabase credentials

# 4. Generate templates
templ generate

# 5. Start development server
air  # Hot reload with Air
# or
go run cmd/server/main.go
```

### Code Guidelines

- Follow Go best practices and formatting (`gofmt`)
- Write tests for new features
- Update documentation for user-facing changes
- Keep commits atomic and well-described
- Run `templ generate` after template changes

### Pull Request Process

1. Create a feature branch from `main`
2. Make your changes with clear commit messages
3. Add tests if applicable
4. Update documentation
5. Submit PR with description of changes
6. Wait for review and address feedback

### Code of Conduct

- Be respectful and inclusive
- Help others learn and grow
- Focus on constructive feedback
- Follow the license terms (Commons Clause)

## License

This project is licensed under the **MIT License with Commons Clause** - see the [LICENSE](LICENSE) file for details.

### What This Means:

✅ **You CAN:**
- Use for personal projects
- Modify and customize the code
- Self-host for yourself or your organization
- Share with others
- Contribute improvements back to the project

❌ **You CANNOT:**
- Sell the software or offer it as a paid service
- Provide hosting services for a fee
- Offer consulting/support services where the primary value is SUDO itself
- Create a competing SaaS product

### Commercial Licensing

If you need to use SUDO commercially or offer it as a service, please don't that defeats the entire pupose of the project.

**Copyright (c) 2025 Paarth Sharma**

---

## Roadmap

### ✅ Completed
- Email-based OTP authentication
- Multi-board kanban interface
- Real-time WebSocket collaboration
- Nested boards for complex tasks
- Dark/Light mode toggle
- Profile management and avatars
- Contact/collaborator management
- Account deletion with data cleanup
- Military-grade encryption (AES-256-GCM)
- Production-ready self-hosting setup

### 🚧 In Progress
- Comprehensive test suite
- Performance optimization

### 📋 Planned
- **File Attachments** - Upload files to tasks
- **Labels & Tags** - Organize tasks with custom labels
- **Filters & Sorting** - Advanced task filtering
- **Keyboard Shortcuts** - Power user navigation
- **Gantt Chart View** - Timeline visualization
- **Notifications** - thinking of browser notifications for updates

Want to contribute to any of these? Check the [Contributing](#contributing) section!

---

## FAQ

**Q: Is SUDO really free?**\
A: Yes! SUDO is source-available and free for personal use and self-hosting. You can deploy it on your own infrastructure at no cost.

**Q: What's the difference between "open source" and "source-available"?**\
A: SUDO uses the Commons Clause license, which allows you to use and modify the code freely, but prohibits selling it as a service. This protects the project from being exploited commercially while keeping it free for everyone.

**Q: Can I use SUDO for my company?**\
A: Yes! You can self-host SUDO for internal company use at no cost. You just cannot offer SUDO as a paid service to others.

**Q: How is my data protected?**\
A: All sensitive data (emails, OTPs) is encrypted using AES-256-GCM with Argon2id key derivation. See [SECURITY.md](SECURITY.md) for details.

**Q: Can I contribute to SUDO?**\
A: Absolutely! We welcome contributions. See the [Contributing](#contributing) section for guidelines.

**Q: Is there a hosted version?**\
A: Yes I host a version for close family and friends. SUDO is designed to be self-hosted for maximum privacy and control.

**Q: What's the difference between using Railway vs self-hosting?**\
A: Railway is easier but costs monthly. Self-hosting gives you full control and can be cheaper long-term. See [SELF_HOST.md](SELF_HOST.md) for comparison.

---

## Acknowledgments

Built with amazing open-source technologies:

- [Go](https://go.dev/) - Backend language
- [Gin](https://gin-gonic.com/) - Web framework
- [HTMX](https://htmx.org/) - HTML-over-the-wire interactivity
- [Templ](https://templ.guide/) - Type-safe Go templating
- [TailwindCSS](https://tailwindcss.com/) - Utility-first CSS
- [Supabase](https://supabase.com/) - PostgreSQL backend
- [Resend](https://resend.com/) - Email delivery

Special thanks to all contributors and the open-source community!

---

## Support

- 📖 [Documentation](SELF_HOST.md)
- 💬 [GitHub Discussions](https://github.com/paarth-sharma/sudo/discussions)
- 🐛 [Issue Tracker](https://github.com/paarth-sharma/sudo/issues)
- 📧 Email: [jobs.paarth@gmail.com]

---

<div align="center">

**[⬆ back to top](#sudo-suck-it-up-and-do-it)**

Made by [Paarth Sharma](https://github.com/paarth-sharma)

If you find SUDO useful, consider giving it a ⭐ on GitHub!

</div>
