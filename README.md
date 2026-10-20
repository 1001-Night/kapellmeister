# Kapellmeister

## Связанные репозитории

- Используемый Helm-чарт - [https://gitlab.com/1001n/kapellmeister-test-chart](https://gitlab.com/1001n/kapellmeister-test-chart)
- Тесты (скрипты + манифесты) - [https://gitlab.com/1001n/kapellmeister-e2e-tests/](https://gitlab.com/1001n/kapellmeister-e2e-tests/)

## Установка

Установка последней версии:
```sh
curl -fsSL https://gitlab.com/1001n/kapellmeister/-/raw/main/install.sh | sh
```

Установка конкретной версии:

```sh
curl -fsSL https://gitlab.com/1001n/kapellmeister/-/raw/main/install.sh | sh -s -- --version v1.2.3
```

Установка без `sudo` (пример):

```sh
mkdir -p "$HOME/.local/bin"
curl -fsSL https://gitlab.com/1001n/kapellmeister/-/raw/main/install.sh | sh -s -- --install-dir "$HOME/.local/bin"
```

В последнем случае каталог `$HOME/.local/bin` должен входить в `PATH`.

## Использование

### Подключение через OpenStack Magnum

Получить kubeconfig по имени или UUID кластера и выполнить команду:

```sh
kapellmeister version --openstack cluster-name --cloud project-name --clouds-yaml /path/to/clouds.yaml
kapellmeister install --openstack cluster-name --cloud project-name -f values.example.yaml
```

`--clouds-yaml` необязателен. Поиск: `OS_CLIENT_CONFIG_FILE`, текущая папка,
`~/.config/openstack/clouds.yaml`, `/etc/openstack/clouds.yaml`.
Профиль задаётся через `--cloud` или `OS_CLOUD`. Если профиль не выбран,
используются `OS_AUTH_URL` и остальные `OS_*`; без них выбирается единственный
профиль из файла. При нескольких профилях нужно указать имя.
OpenRC нужно предварительно загрузить командой `source`, скрипт утилита не исполняет.

```sh
source project-openrc.sh
kapellmeister verify --openstack cluster-name
```

### Команды

Вывести установленную версию Kapellmeister, а так же Kubernetes (если обнаружен)
```sh
kapellmeister version [flags: --kubeconfig, --context]
```

Установить Kapellmeister
```sh
kapellmeister install
    [flags:
        --version string
        -f path/to/values.yaml
        --kubeconfig /path/to/kubeconfig
        --context context-name
        -v 
    ]
```

Обновить Kapellmeister
```sh
kapellmeister update
    [flags:
        --version string
        -f path/to/values.yaml
        --kubeconfig /path/to/kubeconfig
        --context context-name
        -v
    ]
```

Перед установкой, обновлением и откатом утилита проверяет доступность registry образов с машины запуска. Загрузку образов самим кластером затем подтверждает проверка Pod.

Откатить Kapellmeister на предыдущую ревизию
```sh
kapellmeister rollback
    [flags:
        --revision int
        --kubeconfig /path/to/kubeconfig
        --context context-name
        -v
    ]
```

Для отката на конкретную ревизию:
```sh
kapellmeister rollback --revision 2
```

Посмотреть доступные ревизии:
```sh
kapellmeister history
```

Посмотреть состояние последней ревизии по данным Helm:
```sh
kapellmeister status
```

Удалить Kapellmeister
```sh
kapellmeister uninstall
    [flags:
        --kubeconfig /path/to/kubeconfig
        --context context-name
        -v
    ]
```

Проверить наличие Kapellmeister
```sh
kapellmeister verify
    [flags:
        --kubeconfig /path/to/kubeconfig
        --context context-name
    ]
```

Сохранить результат проверки в JSON:
```sh
kapellmeister verify --report-file report.json
```

`--report-file` также работает с `install`, `update` и `rollback`. Если файл уже существует, он заменяется.
Для этих команд отчёт создаётся после запуска проверки; ошибки до неё остаются в журнале утилиты.
