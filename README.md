# 🚀 Auth Service (DDD) – Production-Ready Authentication System in Go

![Go Version](https://img.shields.io/badge/Go-1.22+-blue.svg)
![Architecture](https://img.shields.io/badge/Architecture-DDD%20%7C%20Clean-green)
![Database](https://img.shields.io/badge/Database-SQLite3-lightgrey)
![Status](https://img.shields.io/badge/Status-Production--Ready-success)

---

## 🧠 Overview

This is a **production-grade authentication service** built in **Golang** using **Domain-Driven Design (DDD)** and **Clean Architecture**.

It supports:

* OTP-based authentication
* Email/password login
* MPIN-based login
* Secure password reset flow
* JWT-based authentication system

The project is structured for **scalability, maintainability, and real-world backend engineering**.

---

## 🏗 Project Structure

```bash
AUTHENTICATION/
│
├── cmd/
│   └── api/                 # Application entry point (main.go)
│
├── internal/
│   ├── application/
│   │   └── auth/            # Usecases (business logic)
│   │
│   ├── config/              # Environment configuration (.env loader)
│   │
│   ├── domain/
│   │   └── auth/            # Core business entities & interfaces
│   │
│   ├── infrastructure/
│   │   ├── hash/            # bcrypt hashing
│   │   ├── id/              # ID generation (UUID etc.)
│   │   ├── notify/          # Email/SMS services
│   │   ├── otp/             # OTP generation & verification
│   │   ├── persistence/     # Database (SQLite repositories)
│   │   ├── rate_limiter/    # Rate limiting logic
│   │   └── token/           # JWT token management
│   │
│   ├── interfaces/
│   │   └── http/
│   │       ├── handler/     # HTTP handlers (controllers)
│   │       └── router.go    # Route definitions
│
├── migrations/              # DB migration files
│
├── pkg/
│   └── response/            # Common API response structure
│
├── .env                     # Environment variables
├── auth.db                  # SQLite database
├── go.mod
├── go.sum
├── README.md
```

---

## 🔐 Core Features

### ✅ Authentication Methods

* Email + Password Login
* Mobile + OTP Login
* ClientID + MPIN Login

### 📱 OTP System

* Secure OTP generation (crypto/rand)
* OTP hashing (bcrypt)
* Expiry-based validation
* One-time usage (auto delete)
* Multi-purpose OTP (login + reset password)

### 📌 Mobile OTP Login

```text
Send OTP → Generate OTP Verifiy Token → Verify OTP → Generate OTP Acess Token (rateLimiting) → MPIN Login - Generate Access Token
```

### 📌 Email Login

```text
Email + Password → Generate OTP Acess Token (rateLimiting) → MPIN Login - Generate Access Token
```


### 🔁 Password Reset Flow

```text
Request Reset → Generate OTP Verifiy Token → Send OTP → Verify OTP → Generate Reset Token → Set New Password
```

---

### 🔑 Token System

The authentication system uses **purpose-based JWT tokens** to ensure secure and controlled access across different flows.

#### 🟢 Access Token

* Used for authenticated API requests
* Short-lived (15 minutes)
* Sent in `Authorization: Bearer` header

---

#### 🟡 OTP Verification Token

* Used during OTP verification flow
* Short-lived (5 minutes)
* Ensures OTP is verified for the correct purpose (login/reset)

---

#### 🔵 Reset OTP Verification Token

* Used specifically for password reset OTP verification
* Prevents mixing login OTP and reset OTP flows
* Short-lived (5 minutes)

---

#### 🔴 Reset Password Token

* Issued after successful OTP verification (reset flow)
* Used to authorize password change
* Short-lived (5–10 minutes)
* Should ideally be **one-time use**

---

#### 🟣 (Optional) OTP Access Token

* Temporary token issued after OTP verification (login flow)
* Can be used before full session/token issuance
* Usually avoid unless needed
* Hera apply rate limiting so user cannot try mpin login again and again

---


## ⚙️ Tech Stack

| Layer     | Technology     |
| --------- | -------------- |
| Language  | Go (Golang)    |
| Router    | Gin            |
| Database  | SQLite3        |
| Migration | golang-migrate |
| Auth      | JWT            |
| Hashing   | bcrypt         |
| Config    | godotenv       |

---

## 📦 Setup & Installation

### 1️⃣ Clone Repo

```bash
git clone https://github.com/vishalyadav0987/go-auth-ddd.git
cd go-auth-ddd
```

---

### 2️⃣ Install Dependencies

```bash
go mod tidy
```

---

### 3️⃣ Setup Environment

Create `.env`:

```env
APP_PORT=8069
DB_PATH=./auth.db
JWT_SECRET=supersecretkey
ACCESS_TOKEN_TTL=15m
```

---

### 4️⃣ Run Server

```bash
go run cmd/api/main.go
```

Server runs on:

```
http://localhost:8069
```

---

## 🔐 API Endpoints

### 🔹 Base URL

```http
/api/v1/auth
```

---

# 🆕 🧑‍💻 User Registration

```http
POST /api/v1/auth/register
```

---

# 🔐 Login Flows

## 📧 Login with Email & Password

```http
POST /api/v1/auth/login-with-password
```

---

## 🔐 Login with MPIN

```http
POST /api/v1/auth/login-with-mpin
```

---

## 📱 Login with OTP

### 1️⃣ Send OTP

```http
POST /api/v1/auth/login-with-otp
```

---

### 2️⃣ Verify OTP

```http
POST /api/v1/auth/verify-otp
Authorization: Bearer <otp_verification_token>
```

---

# 🔁 Password Reset Flow

### 1️⃣ Request Reset OTP

```http
POST /api/v1/auth/password/reset/request
```

---

### 2️⃣ Verify Reset OTP

```http
POST /api/v1/auth/password/reset/verify-otp
Authorization: Bearer <otp_reset_verification_token>
```

---

### 3️⃣ Create New Password

```http
POST /api/v1/auth/password/reset/confirm
Authorization: Bearer <reset_password_token>
```

---

## 🔒 Security Highlights

* Password hashing using bcrypt
* OTP hashing (never stored in plain text)
* Token-based authentication (JWT)
* OTP expiry + one-time usage
* Token purpose validation (prevents misuse)
* Ready for rate limiting & brute-force protection

---

## 🚀 Production Enhancements (Planned / Extendable)

* Redis-based OTP storage
* Rate limiting (per user/IP)
* Token blacklist / one-time token usage
* Device/session tracking
* Audit logging
* Microservice migration (gRPC)

---

## 🧠 Design Philosophy

> "Business logic should not depend on frameworks"

✔ Clean Architecture
✔ Domain-Driven Design
✔ Separation of concerns
✔ Scalable structure

---

## 👨‍💻 Author

**Vishal Yadav**

---

## ⭐ Support

If you found this useful, give it a ⭐ on GitHub 🚀
