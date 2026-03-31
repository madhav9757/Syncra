import sys

def split_file(file_path):
    with open(file_path, 'r', encoding='utf-8') as f:
        lines = f.readlines()

    header = "package main\n\nimport (\n\t\"context\"\n\t\"encoding/hex\"\n\t\"encoding/json\"\n\t\"fmt\"\n\t\"math/rand\"\n\t\"os\"\n\t\"path/filepath\"\n\t\"syncra/internal/client/storage\"\n\tclientWS \"syncra/internal/client/websocket\"\n\t\"syncra/internal/config\"\n\t\"syncra/internal/crypto\"\n\t\"syncra/internal/discovery\"\n\t\"syncra/internal/models\"\n\t\"syncra/internal/server/database\"\n\t\"syncra/internal/ui\"\n\t\"time\"\n\n\t\"github.com/charmbracelet/bubbles/spinner\"\n\t\"github.com/charmbracelet/bubbles/textinput\"\n\ttea \"github.com/charmbracelet/bubbletea\"\n\t\"github.com/charmbracelet/lipgloss\"\n)\n\n"

    # model.go
    model_content = header + ''.join(lines[26:80]) + ''.join(lines[182:189]) + ''.join(lines[239:243]) + ''.join(lines[314:318]) + ''.join(lines[351:355])
    with open('model.go', 'w', encoding='utf-8') as f:
        f.write(model_content)

    # commands.go
    commands_content = header + ''.join(lines[190:208]) + ''.join(lines[233:238]) + ''.join(lines[244:313]) + ''.join(lines[319:350]) + ''.join(lines[356:372]) + ''.join(lines[928:947])
    with open('commands.go', 'w', encoding='utf-8') as f:
        f.write(commands_content)

    # update.go
    update_content = header + ''.join(lines[209:232]) + ''.join(lines[373:744])
    with open('update.go', 'w', encoding='utf-8') as f:
        f.write(update_content)

    # view.go
    view_content = header + ''.join(lines[745:915])
    with open('view.go', 'w', encoding='utf-8') as f:
        f.write(view_content)

    # main.go
    main_content = header + ''.join(lines[81:152]) + ''.join(lines[153:181]) + ''.join(lines[916:928])
    with open('main.go', 'w', encoding='utf-8') as f:
        f.write(main_content)

split_file('main.go')
