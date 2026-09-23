# DeviceInventory

Sistema web para gestionar el inventario de dispositivos del proyecto — laptops, desktops, monitores, periféricos, equipos de red, facturación y enlaces WAN.

## Stack

- **Frontend**: React 18 + Vite + TypeScript (modo estricto), Axios, React Router v6
- **Backend**: Go 1.23, chi v5 router, pgx v5 (sin ORM), `log/slog`
- **Base de datos**: PostgreSQL 16 — fechas almacenadas como texto `DD/MM/YYYY`
- **Orquestación**: Docker Compose

## Prerrequisitos

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) instalado y corriendo
- Docker Compose v2 (incluido en Docker Desktop)

## Configuración inicial

```bash
# 1. Copia el archivo de variables de entorno
cp .env.example .env

# 2. Edita .env y ajusta las variables
#    (DB_PASSWORD, JWT_SECRET, SEED_*_PASSWORD)
```

## Levantar el stack

```bash
docker compose up --build
```

Esto inicia tres servicios:

| Servicio | Puerto | Descripción |
|---|---|---|
| `db` | `5432` | PostgreSQL 16 |
| `api` | `8080` | API REST en Go |
| `web` | `5173` | Frontend React (Vite dev server) |

Abre **http://localhost:5173** en tu navegador.

## Usuarios semilla

Al primer arranque se crean automáticamente tres usuarios. Las contraseñas se configuran en `.env`:

| Usuario   | Rol           | Variable en .env          |
|-----------|---------------|---------------------------|
| `admin`   | Administrador | `SEED_ADMIN_PASSWORD`     |
| `manager` | Manager       | `SEED_MANAGER_PASSWORD`   |
| `user`    | Consulta      | `SEED_USER_PASSWORD`      |

> Todos los usuarios semilla tienen `must_change_password = true` — al primer login se forzará cambiar la contraseña.

## Detener el stack

```bash
docker compose down        # detiene contenedores
docker compose down -v     # detiene y elimina volúmenes (borra la BD)
```

## Migraciones de base de datos

El esquema inicial se aplica automáticamente al primer arranque del contenedor `db` desde `db/init/`.  
Las migraciones en `db/migrations/` deben aplicarse **manualmente**:

```bash
docker compose exec db psql -U $DB_USER -d $DB_NAME -f /docker-entrypoint-initdb.d/<migration>.sql
```

## Vistas del sistema

| Ruta | Vista | Acceso |
|---|---|---|
| `/computers` | Inventario unificado (Laptops + Desktops + Monitores) | Todos los roles |
| `/peripherals` | Periféricos (Mouse, Teclado, Headset, Cable) | Todos los roles |
| `/network` | Infraestructura de red (Switch, Router, Firewall, etc.) | Todos los roles |
| `/bios` | Gestión de contraseñas BIOS | Todos los roles |
| `/epd` | EPD — dispositivos agrupados por empleado | Todos los roles |
| `/reports` | Reportes y estadísticas | Todos los roles |
| `/billing` | Facturación | Todos los roles |
| `/enlaces` | Gestión de enlaces WAN/ISP | Todos los roles |
| `/users` | Administración de usuarios | Solo admin |

## Roles

| Rol | Permisos |
|---|---|
| `viewer` | Solo lectura en todas las vistas |
| `manager` | Lectura + crear/editar/eliminar dispositivos, importar/exportar |
| `admin` | Todo lo anterior + gestión de usuarios |

## Importación y exportación de inventario

- **Exportar Excel**: botón en cualquier vista de inventario → descarga un `.xlsx` con los datos de esa vista
- **Descargar plantilla**: botón para obtener la plantilla estándar de importación
- **Importar**: flujo de 2 pasos — *Preview* (análisis sin cambios) → *Apply* (aplica los cambios a la BD)
- Formatos soportados: `.xlsx` y `.csv`
