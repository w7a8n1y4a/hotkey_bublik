# Donut controlling UnitNodes

Parameter | Implementation
-- | --
Description | Позволяет подготавливать и отправлять сообщения в `Input` `UnitNodes` других `Unit`
Lang | `Golang`
Hardware | `Any-PC`
Firmware | `Linux-X11`
Stack | [ebiten-v2](https://github.com/hajimehoshi/ebiten), [pepeunit_go_client](https://github.com/w7a8n1y4a/pepeunit_go_client), [systray](https://github.com/getlantern/systray), [clipboard](https://github.com/atotto/clipboard), [imaging](https://github.com/disintegration/imaging), [screenshot](https://github.com/kbinani/screenshot)
Version | 1.0.0
License | AGPL v3 License
Authors | Ivan Serebrennikov <admin@silberworks.com>

## Env variable assignment

1. `RADIUS_INNER` - Внутренний радиус первого бублика в пикселях
2. `THICK_SEGMENT` - Ширина бубликов в пикселях

## Assignment of Device Topics

- `output_units_nodes/pepeunit` - Предназначен для управления `Input` `UnitNode` других `Unit`, для корректной работы нужно добавить его через `Related Output`. Данные не публикуются в данный топик, а только в связи этого топика

## Work algorithm

1. Чтобы приложение увидело `Unit` на первом слое бублика, нужно соеденить `output_units_nodes/pepeunit` с `Input` другого `Unit` в интерфейсах `Pepeunit`
2. После запуска открыть интерфейс можно или сочетанием клавиш `CTRL + SHIFT + H` или нажав кнопку `Меню` в `system tray`
3. На первом слое бублика каждый сегмент - это отдельный `Unit`
4. На втором слое бублика каждый сегмент - это отдельный `Input` уже выбранного `Unit`
5. На третьем слое бублика каждый сегмент - это сохранённые опции для отправки в выбранный `Input` топик `Unit`
6. При нажатии на `ЛКМ` отображается следующий слой
7. При нажатии на `ПКМ` отображается предидущий слой
8. На третьем слое - при нажатии на `ЛКМ` по `Сreate New Option` будет предложено сначала ввести название опции, после нажатия `Enter` будет предложено ввести данные, которые будут отправлены. Оба поля ввода поддерживают `ctrl-v`
9. На третьем слое - при нажатии на `ЛКМ` по другим сегментам, будут отправляться `MQTT` сообщения в выбранный `Input` топик `Unit`
10. На третьем слое - при нажатии горячей клавиши `Delete` опция будет удалена
11. Опции создаваемые через `Сreate New Option` хранятся на стороне `Pepeunit` через хранилище состояний

## Installation

1. Скачайте бинарный файл последней версии из [releases](https://git.pepemoss.com/pepe/pepeunit/units/go/hotkey_donut/-/releases)
2. Создайте `Unit` в `Pepeunit`
3. Установите переменные окружения в `Pepeunit`
4. Скачайте архив с `env.json` и `schema.json` из `Pepeunit`
5. Запустите бинарный файл
