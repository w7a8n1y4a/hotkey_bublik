# Go Hotkeys

## Description
- Позволяет вводить данные для отправки в связанные `Input` топики.
- Позволяет хранить данные для отправки в связанные `Input` топики.
- Позволяет виборочно публиковать сохранённые команды в связанные `Input` топики.

## Software platform
- [Go v1.23](https://tip.golang.org/doc/go1.23)

## Firmware format
Компилируемый

## Hardware platform
- `PC` - `Linux`

## Required physical components
- `PC`
 
## env_example.json

```json
{   
    "PEPEUNIT_URL": "unit.example.com",
    "PEPEUNIT_APP_PREFIX": "/pepeunit",
    "PEPEUNIT_API_ACTUAL_PREFIX": "/api/v1",
    "HTTP_TYPE": "https",
    "MQTT_URL": "emqx.example.com",
    "MQTT_PORT": 1883,
    "PEPEUNIT_TOKEN": "jwt_token",
    "SYNC_ENCRYPT_KEY": "32_bit_encrypt_key",
    "SECRET_KEY": "32_bit_secret_key",
    "PING_INTERVAL": 30,
    "STATE_SEND_INTERVAL": 300,
    "RADIUS_INNER": 200,
    "THICK_SEGMENT": 50
}

```

### Env variable assignment
1. `RADIUS_INNER` - внутренний радиус первого бублика пикера в пикселях
1. `THICK_SEGMENT` - ширина бубликов в пикселях

## schema_example.json

```json
{
    "input_base_topic": [
        "update/pepeunit",
        "schema_update/pepeunit",
        "env_update/pepeunit"
    ],
    "output_base_topic": [
        "state/pepeunit"
    ],
    "input_topic": [],
    "output_topic": [
        "output_units_nodes/pepeunit"
    ]
}

```

### Assignment of Device Topics
- `output` `output_units_nodes/pepeunit` - используется для организации связей, сюда не публикуются данные.

## Work algorithm
Алгоритм работы с момента нажатия кнопки включения:
1. Подключение к `MQTT Брокеру`
1. Регистрация горячей клавиши вызова - `CTRL + SHIFT + H`
1. Запуск приложения в режиме `Tray`

Алгоритм работы в момент нажатия горячей клавиши или нажатия кнопки `Меню` в `Tray`:
1. Создаётся картинка с размытием по гаусу на весь экран
1. Поверх картинки отображается сегментный бублик
1. На первом слое бублика каждый сегмент - это отдельный `Unit`. 
    - При нажатии на `ЛКМ` отображается следующий слой
1. На втором слое бублика каждый сегмент - это отдельный `Input` уже выбранного `Unit`
    - При нажатии на `ЛКМ` отображается следующий слой
    - При нажатии на `ПКМ` отображается предидущий слой
1. На третьем слое бублика каждый сегмент - это сохранённые опции для отправки в выбранный `Input` топик `Unit`
    - При нажатии на `ЛКМ` по `Сreate New Option` будет предложено сначала ввести название опции, после нажатия `Enter` будет предложено ввести данные, которые будут отправлены. Оба поля ввода поддерживают `ctrl-v`
    - При нажатии на `ЛКМ` по другим сегментам, будут отправляться `MQTT` сообщения в текущий `Input` топик `Unit`
    - При нажатии `Delete` опция будет удалена
    - При нажатии на `ПКМ` отображается предидущий слой

Аспекты работы приложения:
- Чтобы приложение увидело `Unit` на первом слое бублика, нужно соеденить `output_units_nodes/pepeunit` с `Input` другого `Unit` в интерфейсах `Pepeunit`
- Опции создаваемые через `Сreate New Option` хранятся на стороне `Pepeunit` через хранилище состояний