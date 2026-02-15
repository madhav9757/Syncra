<div align="center">
  <img src="static/syncra(logo).png" alt="Syncra Logo" width="180">

# <img src="static/icons/lock.svg" width="32" style="vertical-align: middle; margin-right: 8px;"> Syncra

### The Definitive Technical Blueprint for High-Performance E2EE Communication

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://golang.org)
[![Security: E2EE](https://img.shields.io/badge/Security-E2EE-00b4d8?style=for-the-badge&logo=shield&logoColor=white)](https://en.wikipedia.org/wiki/End-to-end_encryption)
[![Architecture: Zero--Knowledge](https://img.shields.io/badge/Architecture-Zero--Knowledge-ff9f1c?style=for-the-badge&logo=blueprint&logoColor=white)](https://en.wikipedia.org/wiki/Zero-knowledge_proof)
[![Status: Blueprint](https://img.shields.io/badge/Status-Blueprint-green?style=for-the-badge)](https://github.com/)

---

  <p align="center">
    <b>"Assume the Server is Malicious."</b><br>
    <i>This is the founding principle of Syncra. Every line of code in the client is written with the expectation that the relay server is being monitored by a third party.</i>
  </p>

---

</div>

## <img src="static/icons/book.svg" width="24" style="vertical-align: middle; margin-right: 8px;"> Table of Contents

- [<img src="static/icons/layers.svg" width="18" style="vertical-align: middle; margin-right: 4px;"> Infrastructure & Scalability](#%EF%B8%8F-infrastructure--scalability)
- [<img src="static/icons/folder.svg" width="18" style="vertical-align: middle; margin-right: 4px;"> Best-Practice Project Organization](#-best-practice-project-organization)
- [<img src="static/icons/lock.svg" width="18" style="vertical-align: middle; margin-right: 4px;"> Cryptographic Protocol Deep Dive](#-cryptographic-protocol-deep-dive)
- [<img src="static/icons/network.svg" width="18" style="vertical-align: middle; margin-right: 4px;"> The Networking Stack](#-the-networking-stack-websocket--redis)
- [<img src="static/icons/terminal.svg" width="18" style="vertical-align: middle; margin-right: 4px;"> Terminal UI (TUI) Engineering](#-terminal-ui-tui-engineering)
- [<img src="static/icons/database.svg" width="18" style="vertical-align: middle; margin-right: 4px;"> Storage & Local Sovereignty](#-storage--local-sovereignty)
- [<img src="static/icons/checklist.svg" width="18" style="vertical-align: middle; margin-right: 4px;"> Implementation Checklist](#%EF%B8%8F-implementation-checklist-step-by-step)
- [<img src="static/icons/shield.svg" width="18" style="vertical-align: middle; margin-right: 4px;"> Security Threat Model](#%EF%B8%8F-security-threat-model)

---

## <img src="static/icons/layers.svg" width="24" style="vertical-align: middle; margin-right: 8px;"> Infrastructure & Scalability

### <img src="static/icons/folder.svg" width="22" style="vertical-align: middle; margin-right: 6px;"> Codebase Lifecycle & Project Anatomy

The project follows a **Modular Monorepo** approach, separating the high-concurrency relay logic from the cryptographic heavy-lifting of the client.

```text
syncra/
├── 📂 cmd/                         # Entry points (Source of Truth)
│   ├── 📂 server/                  # The Stateless Blind Relay
│   │   └── main.go
│   └── 📂 cli/                     # The Terminal CLI Suite
│       └── main.go
│
├── 📂 internal/                    # Private Application Core
│   ├── 📂 server/                  # Server-side Business Logic
│   │   ├── 📂 websocket/           # Hub, Handler, & Socket Lifecycle
│   │   ├── 📂 auth/                # JWT Proofs & Challenge-Response
│   │   ├── 📂 database/            # PostgreSQL Repository Layer
│   │   ├── 📂 redis/               # Real-time Pub/Sub Orchestrator
│   │   ├── 📂 router/              # HTTP/WSS Routing Entry
│   │   └── 📂 service/             # Domain Logic & User Orchestration
│   │
│   ├── 📂 client/                  # Client-side Business Logic
│   │   ├── 📂 websocket/           # Outbound Connection Lifecycle
│   │   ├── 📂 crypto/              # AES-256-GCM & Ed25519 Pipelines
│   │   ├── 📂 storage/             # Local-First Filesystem Engine
│   │   ├── 📂 auth/                # Identity Proof Generation
│   │   ├── 📂 ui/                  # Bubble Tea Reactive Components
│   │   └── 📂 config/              # User Preference Management
│   │
│   ├── 📂 shared/                  # Agnostic Contracts & Models
│   │   ├── 📂 models/              # Cross-boundary Structs
│   │   ├── 📂 dto/                 # Packet Transfer Objects
│   │   └── 📂 constants/           # Shared Event Definitions
│   │
│   └── 📂 pkg/                     # Reusable Tooling
│       ├── 📂 logger/              # Structured Zero-Log
│       └── 📂 utils/               # High-performance Helpers
│
├── 📂 deployments/                 # Orchestration & Ingress
│   ├── 📂 docker/                  # Multi-stage Docker Builds
│   └── 📂 nginx/                   # Reverse Proxy & SSL Config
│
├── 📂 scripts/                     # Automation & CI/CD
└── 📂 configs/                     # Environment Variable Schemas
```

> [!TIP]
> **<img src="static/icons/bulb.svg" width="16" style="vertical-align: middle; margin-right: 4px;"> Senior Mindset**: By keeping the `shared/` folder minimal, we ensure that the client binary strictly contains zero server-side vulnerabilities, and the server strictly contains zero cryptographic keys.

---

### <img src="static/icons/globe.svg" width="24" style="vertical-align: middle; margin-right: 8px;"> High-Level Topology

The backend isn't just a single server; it's a **stateless relay cluster**.

<div align="center">
  <img src="https://img.shields.io/badge/Relay--Cluster-Stateless-blue?style=for-the-badge" alt="Stateless Relay Cluster">
</div>

```text
══════════════════════════════════════════════════════════════════════════════════════
                          PROJECT INFRASTRUCTURE TOPOLOGY
══════════════════════════════════════════════════════════════════════════════════════

      [ PUBLIC TIER ]          [ INGRESS CONTROL ]          [ BACKEND CLUSTER ]

      ┌────────────┐          ┌───────────────────┐        ┌──────────────────────┐
      │  User App  │ ════════▶│  Nginx / Caddy   │ ═══════▶│  Relay Nodes (xN)   │
      │  (Go CLI)  │ :443/WSS │  (SSL Terminate) │ :50051  │  (Stateless Go)     │
      └────────────┘          └─────────┬─────────┘        └──────────┬───────────┘
                                        │                             │
                                        ▼                             ▼
                                ┌──────────────────┐          ┌──────────────────┐
                                │ Cloudflare/WAF   │          │  Redis Pub/Sub   │
                                │ (DDoS Protect)   │ :6379    │  (Real-time Bus) │
                                └──────────────────┘          └──────────┬───────┘
                                                                         │
                                         ┌──────────────────────────────┘
                                         │
                                         ▼
                                ┌──────────────────┐          ┌──────────────────┐
                                │  PostgreSQL DB   │ :5432    │  JWT / Auth API  │
                                │ (Handles/Public) │◀────────▶│ (Identity Sync)  │
                                └──────────────────┘          └──────────────────┘

══════════════════════════════════════════════════════════════════════════════════════
```

#### 🛠️ Infra Decisions:

- **WSS (WebSocket Secure)**: We use `:443` to bypass 99% of corporate firewalls that block non-standard ports.
- **Redis Pub/Sub**: Crucial for horizontal scaling. If `User A` is on `Node 1` and `User B` is on `Node 2`, Redis acts as the glue that routes the encrypted packet between nodes.
- **Statelessness**: The relay nodes store **zero** conversational state.

---

## <img src="static/icons/lock.svg" width="24" style="vertical-align: middle; margin-right: 8px;"> Cryptographic Protocol Deep Dive

### 1. Identity Generation

Upon the first boot, the client generates a permanent identity using **Ed25519**.

- **Reasoning**: Ed25519 provides 128-bit security with tiny 32-byte public keys and 64-byte private keys.
- **Checklist**:
  - [ ] <img src="static/icons/checklist.svg" width="16" style="vertical-align: middle;"> Use `crypto/ed25519` to generate `Seed`, `PublicKey`, and `PrivateKey`.
  - [ ] <img src="static/icons/checklist.svg" width="16" style="vertical-align: middle;"> Encrypt the `PrivateKey` locally using **AES-256-KWP**.

### 2. The Hybrid Encryption Pipeline (E2EE)

Syncra uses a **Sign-then-Encrypt** approach for maximum security.

1.  **Ephemeral Key Gen**: Random 32-byte **AES-256** key for every message.
2.  **Signing**: User A signs the plaintext hash using their **Ed25519 Private Key**.
3.  **Symmetric Encryption (AEAD)**: Plaintext + signature encrypted using **AES-256-GCM**.
4.  **Asymmetric Wrapping**: AES key encrypted using **User B's Public Key**.

---

## <img src="static/icons/network.svg" width="24" style="vertical-align: middle; margin-right: 8px;"> The Networking Stack (WebSocket + Redis)

### 1. The Relay "Hub" Pattern

In Go, the server should run a central `Hub` goroutine that manages active clients.

```go
type Hub struct {
    clients    map[string]*Client // map[UserID]*Client
    broadcast  chan []byte        // Inbound packets
    register   chan *Client       // New connections
    unregister chan *Client       // Dropped connections
}
```

- **Concurrency Tip**: Use **buffered channels** for the `broadcast` channel to prevent a "slow consumer" blocking the relay.

---

## <img src="static/icons/terminal.svg" width="24" style="vertical-align: middle; margin-right: 8px;"> Terminal UI (TUI) Engineering

Powered by [Charmbracelet](https://charm.sh/): `Bubble Tea`, `Lip Gloss`, and `Bubbles`.

- **Model**: Holds the application state.
- **Update**: Handles incoming **Messages**.
- **View**: Renders the state into a string.

---

## <img src="static/icons/database.svg" width="24" style="vertical-align: middle; margin-right: 8px;"> Storage & Local Sovereignty

### 1. Append-Only Chat Logs

Instead of a database, we use flat files for speed and portability.

- **Path**: `~/.syncra/chats/<contact_id>.log`
- **Format**: JSONL (JSON Lines).

---

## <img src="static/icons/checklist.svg" width="24" style="vertical-align: middle; margin-right: 8px;"> Implementation Checklist (Step-by-Step)

<details>
<summary><b><img src="static/icons/layers.svg" width="18" style="vertical-align: middle; margin-right: 4px;"> Phase 1: The Secure Sandbox</b></summary>

- [ ] Initialize Go module: `go mod init github.com/username/syncra`.
- [ ] Implement `crypto/identity.go`: Key generation.
- [ ] Implement `crypto/aes_gcm.go`: AES wrappers.
</details>

<details>
<summary><b><img src="static/icons/network.svg" width="18" style="vertical-align: middle; margin-right: 4px;"> Phase 2: The Blind Messenger (Server)</b></summary>

- [ ] Setup `net/http` with `gorilla/websocket`.
- [ ] Create the `Hub` and `Client` structures.
- [ ] Implement Redis `PUBLISH/SUBSCRIBE` loop.
</details>

<details>
<summary><b><img src="static/icons/terminal.svg" width="18" style="vertical-align: middle; margin-right: 4px;"> Phase 3: The Terminal Experience (Client)</b></summary>

- [ ] Scaffold `Bubble Tea` program.
- [ ] Add an "Onboarding" screen.
- [ ] Implement the message history viewport.
</details>

---

## <img src="static/icons/shield.svg" width="24" style="vertical-align: middle; margin-right: 8px;"> Security Threat Model

| Threat                | Prevention / Mitigation                                                     |
| :-------------------- | :-------------------------------------------------------------------------- |
| **Server Inspection** | Messages are encrypted at the edge. Server only sees metadata.              |
| **Identity Forgery**  | All messages are signed with User A's Private Key.                          |
| **Traffic Replay**    | Unique `Nonce` and `Timestamp`; old packets are discarded.                  |
| **Server Compromise** | Server is stateless and blind; attacker gains nothing but current metadata. |

---

<div align="center">

### <img src="static/icons/flag.svg" width="24" style="vertical-align: middle; margin-right: 8px;"> Professional Engineering Narrative

> "I architected a high-concurrency, Zero-Knowledge messaging system. The core innovation is the separation of the **Identity Layer** (Ed25519) from the **Transport Layer** (WebSockets/Redis). By offloading all decryption logic to the Go binary, I ensured that the backend serves only as a 'Blind Relay', maintaining absolute privacy even in the event of breaches."

</div>

---

## <img src="static/icons/link.svg" width="24" style="vertical-align: middle; margin-right: 8px;"> Deep Learning Links

<p align="center">
  <a href="https://nostarch.com/seriouscrypto">Cryptography</a> •
  <a href="https://www.youtube.com/watch?v=0Zbh_S_28sc">Go Concurrency</a> •
  <a href="https://guide.elm-lang.org/architecture/">TUI Design</a> •
  <a href="https://redis.io/docs/manual/pubsub/">Distributed Systems</a>
</p>
