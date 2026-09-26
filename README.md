# Kapellmeister

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
kapellmeister version [flags: --kubeconfig] 
```

Установить Kapellmeister
```sh
kapellmeister install
    [flags:
        --version string
        -f path/to/values.yaml
        --kubeconfig /path/to/kubeconfig
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
        -v
    ]
```

Удалить Kapellmeister
```sh
kapellmeister uninstall
    [flags:
        --kubeconfig /path/to/kubeconfig
        -v
    ]
```

Проверить наличие Kapellmeister
```sh
kapellmeister verify
    [flags:
        --kubeconfig /path/to/kubeconfig
    ]
```