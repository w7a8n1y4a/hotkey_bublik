# Hotkey Bublik

Parameter | Implementation
-- | --
Description | Управляет командами, отправляемыми в `Input` `UnitNodes` других `Unit`. Позволяет вызывать команду выбором сегмента  и/или назначать для команд горячие клавиши
Lang | `Golang`
Hardware | `Any-PC`
Firmware | `Linux`, `X11`, `I3WM`
Stack | [ebiten-v2](https://github.com/hajimehoshi/ebiten), [pepeunit_go_client](https://github.com/w7a8n1y4a/pepeunit_go_client)
Version | 1.0.0
License | AGPL v3 License
Authors | Ivan Serebrennikov <admin@silberworks.com>

## Example

<div align="center"><img align="center" src="https://minio.pepemoss.com/public-data/gif/hotkey_bublik.gif"></div>

## Env variable assignment

1. `RADIUS_INNER` - Внутренний радиус первого бублика в пикселях
2. `THICK_SEGMENT` - Толщина сегментов бубликов в пикселях
3. `HOTKEY_MAIN` - Глобальный хоткей запуска интерфейса (например: `CTRL+SHIFT+P`). Если указать `null` — хоткей не регистрируется

## Assignment of Device Topics

- `output_units_nodes/pepeunit` - Предназначен для управления `Input` `UnitNode` других `Unit`, для корректной работы нужно добавить его через `Related Output`. Данные не публикуются в данный топик, а только в связи этого топика

## Work algorithm

1. Чтобы приложение увидело `Unit` на первом слое бублика, нужно соеденить `output_units_nodes/pepeunit` с `Input` другого `Unit` в интерфейсах `Pepeunit`
2. После запуска открыть интерфейс можно нажав кнопку `Меню` в `system tray` или через хоткей указанный в переменной `HOTKEY_MAIN`
3. На первом слое бублика каждый сегмент - это отдельный `Unit`
4. На втором слое бублика каждый сегмент - это отдельный `Input` `UnitNode` уже выбранного `Unit`
5. На третьем слое бублика каждый сегмент - это сохранённые команды для отправки в выбранный `Input` `UnitNode`
6. Возможные действия с сегментами подписаны под бубликами
7. Команды создаваемые на третьем слое бублика хранятся на стороне `Pepeunit` через хранилище состояний в шифрованом виде
8. В логах на экране отображаются последние `8` записей из `log.json`

## Installation

1. Скачайте бинарный файл последней версии из [releases](https://git.pepemoss.com/pepe/pepeunit/units/go/hotkey_donut/-/releases)
2. Создайте `Unit` в `Pepeunit`
3. Установите переменные окружения в `Pepeunit`
4. Скачайте архив с `env.json` и `schema.json` из `Pepeunit`
5. Запустите бинарный файл
