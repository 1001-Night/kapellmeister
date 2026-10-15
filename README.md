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

Вывести установленную версию Kapellmeisterctl, а так же Kubernetes (если обнаружен)
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
