### **Development Workflow**

*   to make simple, always reuse and centralize code as you can, avoid duplicate code, and overengineering.
*   **Standardize on Docker:** Use `docker` for all package management and `docker` / `docker exec` for script execution (replaces NPM/NPX).
*   **Standardize on Go & Bun :** Use go & typescript (bun) for this project.
*   **Standardize on Makefile:** Use `Makefile` for all package management and `Makefile` / `Makefile exec` for script execution on local or servers.
*   **Centralized Commands:** All development tasks (linting, type-checking, testing, building, deploying) must be executed via `Makefile` targets. Do not invent commands, to ensure consistency. you can run make command on server too.
*   **Leverage MCP:** Actively use the MCP toolset to accelerate development progress (because we use alot last version tech, dont hallusinate to use the feature, find out on official docs if you have knowledge cutoff, and dont use the feature if its not available on the version we use)
* becareful we have submodules, dont change the `signoz` if not nessessary changes, and exclude script/ci/make (lint, test, typecheck, etc) from the submodule.

### **Interaction & Research**

* this project deploys via Docker Compose. Access credentials and host details are stored in the private runbook—use the documented non-root deploy user and SSH keys (no passwords).
* if theres any database problem, dont directly change on server, but must through sql files (edit or create new) on `database/migrations` to ensure consistency when rebuild (theres docker migration will run automatically when rebuild).
* if theres any problem with the server, you can check the logs on `docker logs -f <container_name>` via the documented SSH host alias in the runbook.
* if theres any problem with the database, you can check the logs on `docker logs -f postgres` via the documented SSH host alias in the runbook.
*   **Verify, Don't Assume:** Always find the latest/correct version of documentation and best practices by consulting `context7` or search on web, leverage other MCP tools, and official web resources.

### **Reference Locations**

*   **Legacy Documentation:** `docs/old-docs/`