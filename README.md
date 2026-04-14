# vk-investment-middleend

Middleend (BFF) — serves SDUI component trees to frontends.

## Setup

```bash
cp .env.example .env
go mod tidy
```

## Commands

| Command      | Description                    |
|--------------|--------------------------------|
| `make run`   | Start the server locally       |
| `make build` | Build the binary               |
| `make test`  | Run all tests (JSON output)    |
| `make lint`  | Run linting checks             |
| `make health`| Check application health       |
| `make info`  | Print project metadata         |
| `make clean` | Remove build artifacts         |
