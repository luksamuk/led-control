from machine import Pin
from neopixel import NeoPixel
from time import sleep, ticks_ms
from picozero import pico_led
import network
import socket
import select
import struct

# Definitions for pins
pin = Pin(28)
button = Pin(3, Pin.IN, Pin.PULL_DOWN)

# Definitions for NeoPixel LEDs
np = NeoPixel(pin, 30)
programming = 0

# Definitions for WiFi
SSID   = 'CHANGEME'
PASSWD = 'CHANGEME'
wlan   = network.WLAN(network.STA_IF)

# Definitions for REST server
rest_socket = None
rest_poller = None

# Global state
blinking = True
pressed_old = False
pressed_current = False
last_blink_ms = 0
dim = 0.05
prog2_color = (255, 255, 255)

# All pixel colors which cycle around
# the strip sequentially
pixels_prog0 = [
    (255, 255, 255),
    (0, 0, 0),
    (64,  30,  10),
    (0,    0,   0),
    (30,  64,  10),
    (0,    0,   0),
    (30,  10,  64),
    (0,    0,   0),
    (128, 10,  30),
    (0,    0,   0),
    (10, 128,  30),
    (0,    0,   0),
    (10,  30, 128),
    (0,    0,   0),
    (255, 30,  30),
    (0,    0,   0),
    (30, 255,  30),
    (0,    0,   0),
    (30,  30, 255),
    (0,    0,   0),
    (255, 0, 0),
    (0,    0,   0),
    (255, 165, 0),
    (0,    0,   0),
    (255, 255, 0),
    (0,    0,   0),
    (0, 128, 0),
    (0,    0,   0),
    (0, 0, 255),
    (0,    0,   0),
    (75, 0, 130),
    (0,    0,   0),
    (238, 130, 238),
    (0,    0,   0)
]


# Routine for pressing the button
def pressed():
    global button
    global pressed_old
    global pressed_current
    pressed_current = button.value()
    state = False
    if not (pressed_current == pressed_old):
        if pressed_current:
            state = True
    pressed_old = pressed_current
    return state

# Shut lights off
def lights_off():
    global np
    for i in range(np.n):
        np[i] = (0, 0, 0)
    np.write()

# Change lights dim
def set_dim(dimstr):
    global dim
    print(f'set dim: {dimstr}')
    try:
        deltadim = float(dimstr)
        if (deltadim > 0) and (deltadim <= 1):
            dim = deltadim
    except:
        pass

# Change lantern program color
def set_color(colorstr):
    global prog2_color
    print(f'set color: {colorstr}')
    try:
        prog2_color = struct.unpack('BBB', colorstr.decode('hex'))
    except:
        pass

def set_program(programstr):
    global programming
    try:
        prog = int(programstr)
        if (prog >= 0) and (prog < 3):
            programming = prog
            lights_off()
    except:
        pass
    
# Routine for toggling state
def toggle():
    global blinking
    blinking = not blinking
    if not blinking:
        lights_off()

# Routine for changing the program
def cycle_program():
    set_program((programming + 1) % 3)
        
# Connect to the given WiFi network
def wlan_connect(ssid, password):
    wlan.active(True)
    print('Scanning for networks...')
    wlan.scan()
    print(f'Connecting to {ssid}...')
    wlan.connect(ssid, password)
    while not wlan.isconnected():
        pico_led.toggle()
        sleep(0.2)
    ip = wlan.ifconfig()[0]
    print(f'Connected. IP address: {ip}')
    pico_led.on()
    return ip

# Create a REST socket. Returns a poller and the socket itself.
def start_rest_socket(ip):
    address = (ip, 80)
    rest_socket = socket.socket()
    rest_socket.bind(address)
    rest_socket.listen(1)
    poller = select.poll()
    poller.register(rest_socket, select.POLLIN)
    print(f'Socket listening to {address[0]}:{address[1]}.')
    return (poller, rest_socket)

# Functions for REST responses
def respond_status(client):
    global blinking
    global dim
    global programming
    global prog2_color
    value = 'true' if blinking else 'false'
    color = struct.pack('BBB', *prog2_color).encode('hex')
    client.send('HTTP/1.1 200 OK\r\n')
    client.send('Content-Type: application/json\r\n')
    client.send('Connection: close\r\n')
    client.send(f'\n\r{{"blinking": {value}, "program": {programming}, "dim": {dim}, "color": {color}}}\r\n')

def respond_notfound(client):
    client.send('HTTP/1.1 404 Not Found\r\n')
    client.send('Connection: close\r\n')

# Poll and respond to REST events
def poll_rest_event():
    global rest_poller
    global blinking
    global programming
    res = rest_poller.poll(16)
    if res:
        client = res[0][0].accept()[0]
        request = client.recv(1024).decode('utf-8')
        # Get first line
        request = request.partition('\r\n')[0]
        print(request)
        if request.startswith('POST /led/toggle '):
            toggle()
            respond_status(client)
        elif request.startswith('POST /led/change '):
            cycle_program()
            respond_status(client)
        elif request.startswith('POST /led/on '):
            blinking = True
            respond_status(client)
        elif request.startswith('POST /led/off '):
            blinking = False
            lights_off()
            respond_status(client)
        elif request.startswith('POST /led/dim/'):
            trimright = str(request)[:-9]
            set_dim(trimright[14:])
            respond_status(client)
        elif request.startswith('POST /led/color/'):
            trimright = str(request)[:-9]
            set_color(trimright[16:])
            respond_status(client)
        elif request.startswith('POST /led/program/'):
            trimright = str(request)[:-9]
            set_program(trimright[18:])
            respond_status(client)
        elif request.startswith('GET /led '):
            respond_status(client)
        else:
            respond_notfound(client)
        client.close()

# INDEX
i = 0

# Constant white light programming
def programming2():
    global np
    global blinking
    global dim
    if blinking:
        px = ((int)(dim * prog2_color[0]), (int)(dim * prog2_color[1]), (int)(dim * prog2_color[2]))
        for i in range(np.n):
            np[i] = px
            np.write()

# Blinking LED programmings
def programming0():
    global i
    global blinking
    global np
    global last_blink_ms
    global pixels_prog0
    global dim
    
    num_pixels = len(pixels_prog0)
    current_time_ms = ticks_ms()
    if (current_time_ms - last_blink_ms >= 100): # Blink every 100ms
        if blinking:
            i = (i + 1) % num_pixels
            for j in range(np.n):
                color = pixels_prog0[(i + j) % num_pixels]
                np[j] = ((int)(dim * color[0]), (int)(dim * color[1]), (int)(dim * color[2]))
            np.write()
        last_blink_ms = current_time_ms


# Back-and-forth trail of lights
going_back = False
current_color = 0
def programming1():
    global i
    global blinking
    global np
    global last_blink_ms
    global going_back
    global pixels_prog0
    global current_color
    global dim
    current_time_ms = ticks_ms()
    if (current_time_ms - last_blink_ms >= 20): # Blink every 16ms
        if blinking:
            if ((not going_back) and (i == np.n - 1)) or (going_back and (i == 0)):
                going_back = not going_back
                current_color = (current_color + 2) % len(pixels_prog0)
            if not going_back:
                i = (i + 1) % np.n
                j = (i - 1) % np.n
            elif going_back:
                i = (i - 1) % np.n
                j = (i + 1) % np.n
            #np[i] = (255, 255, 255)
            color = pixels_prog0[current_color]
            np[i] = ((int)(dim * color[0]), (int)(dim * color[1]), (int)(dim * color[2]))
            np[j] = (0, 0, 0)
            np.write()
        last_blink_ms = current_time_ms

# Blinking lights loop, should be spawned asynchronously
def blink_lights_loop():
    global programming
    while True:
        sleep(0.001) # 1ms
        if programming == 0:
            programming0()
        elif programming == 1:
            programming1()
        elif programming == 2:
            programming2()
        else:
            lights_off()
        # Toggle LED if button was pressed
        if pressed():
            toggle()
        # Execute REST events
        poll_rest_event()

if __name__ == "__main__":
    try:
        # Turn lights off if any
        lights_off()
    
        # Light on onboard LED
        pico_led.on()
    
        # Connect to WiFi
        ip = wlan_connect(SSID, PASSWD)
    
        # Start REST server
        rest_poller, rest_socket = start_rest_socket(ip)
    
        # Run on non-interpreter thread
        print('Starting main loop')
        blink_lights_loop()
    except:
        # On exceptions, restart RPi Pico W.
        # This avoids errors such as Address in Use for sockets.
        machine.reset()
