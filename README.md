# LED Control

Iluminação IoT usando LEDs NeoPixel, controle remoto e mudança automática de iluminação.

A intenção original era criar um sistema simples de iluminação dinâmica para o meu quarto, usando IoT, e também permitir controlar isso remotamente (já que não tenho interruptor próximo à minha cama...), além de mudar automaticamente a iluminação em certos momentos do dia.

## MQTT Broker

O primeiro passo é realizar o *deploy* de um broker MQTT, que receberá mensagens de controle e fará broadcast para o Raspberry Pi Pico W. Eu recomendo o projeto [Eclipse Mosquitto](https://mosquitto.org/).

Pessoalmente, tenho um Raspberry Pi 4, que utilizo para hospedar todo tipo de serviço. Para facilitar esse processo, faço uso de uma variação de [Kubernetes](https://kubernetes.io/) chamada [K3s](https://k3s.io/), que foi feita especialmente para IoT e Edge Computing.

### Gerando um usuário e senha para o Mosquitto

O usuário padrão do Mosquitto que preparei é `admin`, e sua senha também é `admin`. Caso você queira trocar essas credenciais, execute os seguintes passos:

1. Certifique-se de que o [Docker](https://www.docker.com/) esteja instalado no seu computador. Execute um console Linux usando a imagem do Eclipse Mosquitto com o seguinte comando:

```bash
docker run -it eclipse-mosquitto:latest /bin/sh
```

2. No console que abre, execute os seguintes comandos para gerar um novo usuário e uma nova senha.

```bash
mosquitto_passwd -c -b password.txt SEUUSUARIO SUASENHA
cat password.txt
exit
```

3. Se você estiver realizando deploy no Kubernetes como eu, copie o que for escrito na tela pelo segundo comando, e cole no arquivo `deploy/mosquitto/mosquitto-pw.yml`, substituindo as credenciais que já estão lá. **LEMBRE-SE, NUNCA ENVIE ESSAS CREDENCIAIS PARA O GITHUB, EM HIPÓTESE ALGUMA!**

### Realizando deploy no K3s

Para realizar deploy no K3s, certifique-se de que você pode acessar seu cluster usando o comando `kubectl`. Também certifique-se de que seu cluster possui um IP que seja acessível por dispositivos na sua rede -- no meu caso, meu Raspberry Pi 4, que está executando o K3s, opera com um IP fixo no meu roteador.

Crie um namespace chamado `iot` (se não quiser esse nome, terá que alterar os arquivos de configuração). Aplique as alterações através do arquivo `kustomization.yml`:

```bash
kubectl apply -k deploy/mosquitto
```

Seu broker MQTT deverá estar acessível a partir desse momento, no IP do seu dispositivo, sob a rota `/mqtt`. Caso você precise de uma aplicação para testar e interagir, recomendo o [MQTTX](https://mqttx.app/).


## Cliente no Raspberry Pi Pico W

![Exemplo do circuito de LEDs em execução](./img/neopixel2.gif)

Vamos usar o Raspberry Pi Pico W como um cliente MQTT e também como um controlador para os LEDs NeoPixel. Esse dispositivo possui uma antena Wi-Fi, por isso ele é ideal para um servidor HTTP IoT.

### Circuito

Os requisitos mínimos para montar o projeto são:

- Uma protoboard ([uma de 400 pontos](https://www.makerhero.com/produto/protoboard-400-pontos/) é o suficiente);
- Um [Raspberry Pi Pico W](https://www.makerhero.com/produto/raspberry-pi-pico-w/) (deve ser este modelo, para que conectemos ao Wi-Fi -- lembre-se de soldar [pinos](https://www.makerhero.com/produto/barra-de-pinos-1x40-torneada-180-graus/) em ambas as extremidades para conectar à protoboard!);
- 7 [jumpers](https://www.makerhero.com/produto/kit-jumpers-10cm-x120-unidades/) macho-macho (para montar o circuito. Três desses jumpers são saídas para os LEDs, e podem ser substituídos por jumpers macho-fêmea a depender da extremidade usada);
- Uma [Fita de LED RGB WS2812 5050 de 1 metro](https://www.makerhero.com/produto/fita-de-led-rgb-ws2812-5050-1m/);
- Uma [chave táctil push-button](https://www.makerhero.com/produto/chave-tactil-push-button/).

A montagem pode ser feita de acordo com essa [esquemática](./img/breadboard.png).


### Instalação

O projeto encontra-se na pasta `rpi-pico-w`.

1. [Baixe a IDE Thonny](https://thonny.org/).
2. [Conecte seu Raspberry Pi Pico W e instale o MicroPython](https://projects.raspberrypi.org/en/projects/get-started-pico-w/1).
3. Instale as dependências. Isso pode ser feito de duas formas:
   - No Thonny, vá no menu `Ferramentas` > `Gerenciar pacotes`. Instale os pacotes `picozero`, `neopixel` e `umqtt.simple`; OU
   - Abra o script `install-modules.py`, altere a chamada `wlan.connect(...)` para usar as credenciais do seu Wi-Fi, e execute esse script no seu Raspberry Pi Pico W.
4. Abra o arquivo `main.py`. Altere as variáveis `SSID` e `PASSWD` para as suas credenciais do seu Wi-Fi.
5. No mesmo arquivo, altere as variáveis `BROKER`, `BROKER_USER` e `BROKER_PASS` para o IP, o usuário e a senha do seu broker MQTT.
6. Você poderá executar a aplicação para testar nesse momento. Para fazer com que o MicroPython fique responsivo novamente, basta parar a aplicação.
7. Após testar, selecione a opção para salvar o script, e salve-o no seu Raspberry Pi Pico W, *exatamente com o nome `main.py`*.

O Raspberry Pi Pico W não precisa ter um IP fixo.

### Comportamento e tópicos

O Raspberry Pi Pico W conecta-se ao broker MQTT e escuta mensagens em tópicos cujo nome seja iniciado com `led/`.

Os tópicos escutados são:

- `led/active`: Interage com o status de ligado ou desligado dos LEDs, através dos números `0` (desligado) ou `1` (ligado).
- `led/program`: Interage com a programação atual das luzes, sendo elas os números:
  - `0`: Luzes de Natal.
  - `1`: Rastro.
  - `2`: Luz fixa (com cores customizáveis).
- `led/dim`: Interage com o *dimmer*, que manipula a intensidade da luz. Espera-se um número entre `0.02` e `1`.
- `led/color`: Interage com a cor da luz fixa, quando o programa `2` estiver sendo utilizado. A cor deve ser um valor hexadecimal RGB, no formato `ffffff`.

É possível mudar e ler as informações nesses tópicos através do broker. O único tópico no qual o Raspberry Pi Pico W publica alguma coisa é o `led/active`, já que o mesmo possui um *push button* que serve como interface física para ligar e desligar as luzes. Mesmo assim, lembre-se de que absolutamente todos o estado dos LEDs está condicionado única e exclusivamente às informações que o broker recebe.

Para garantir que o dispositivo receba novamente o estado que possuía antes de ser desligado, caso seja removido da tomada, envie mensagens para estes tópicos com a *flag* `Retain` sempre ativada.

## Controle Remoto

**ATENÇÃO: ESTA SEÇÃO ENCONTRA-SE OBSOLETA E SERÁ MUDADA ASSIM QUE O CONTROLE REMOTO USAR O PROTOCOLO MQTT.**

![Controle remoto para o servidor HTTP](./img/controle.png)

O controle remoto foi projetado para que seja possível controlar o servidor de LEDs remotamente, interagindo via requisições HTTP, especialmente pelo celular Android, mas também pode ser utilizado no desktop ou em outros sistemas. Ele funcionará normalmente caso você esteja conectado à mesma rede do seu Raspberry Pi Pico W.

### Testando a aplicação

Para compilar a aplicação, você precisa ter o compilador de Go 1.21.0 ou superior. Para compilar para o Android, instale Docker e a ferramenta [`fyne-cross`](https://github.com/fyne-io/fyne-cross).

Comece abrindo o arquivo `internal/wsclient/wsclient.go`. Altere a constante `BASEURL` para o IP fixo do seu Raspberry Pi Pico W.

Em seguida, para testar a aplicação, execute:

```bash
go run .
```

### Compilando para Android

Você poderá compilar sua aplicação para Android entrando na pasta `LEDControl` e executando o comando `make`, caso você tenha o GNU Makefile instalado. Caso contrário, use diretamente o `fyne-cross`:

```bash
fyne-cross android \
    -app-id com.luksamuk.ledcontrol \
    -icon Icon.png
```

*NOTA:* Para o Go 1.21.0, em 16/10/2023, a compilação com `fyne-cross` está quebrada. Recomendo instalar o `fyne-cross`, e instalar Go 1.20 [através da ferramenta GVM](https://github.com/moovweb/gvm).

## Serviço Web

Documentação em breve.

