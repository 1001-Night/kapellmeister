# Kapellmeister

## Установка

Сейчас поддерживается Linux x86_64. Последний релиз устанавливается одной командой:

```sh
curl -fsSL https://gitlab.com/1001n/kapellmeister/-/raw/main/install.sh | sh
```

Установка конкретной версии:

```sh
curl -fsSL https://gitlab.com/1001n/kapellmeister/-/raw/main/install.sh | sh -s -- --version v1.2.3
```

Установка без `sudo`, например в пользовательский каталог:

```sh
mkdir -p "$HOME/.local/bin"
curl -fsSL https://gitlab.com/1001n/kapellmeister/-/raw/main/install.sh | sh -s -- --install-dir "$HOME/.local/bin"
```

В последнем случае каталог `$HOME/.local/bin` должен входить в `PATH`.

Установщик проверяет SHA-256 перед копированием бинарника. Пока в проекте нет ни одного
GitLab Release, он использует артефакт последней успешной сборки ветки `main`.

## Публикация релиза

Релиз и его установочные файлы публикуются CI автоматически при создании semver-тега:

```sh
git tag v1.0.0
git push origin v1.0.0
```
