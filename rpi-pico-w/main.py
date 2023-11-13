from machine import Pin
from neopixel import NeoPixel
from time import sleep, ticks_ms
from picozero import pico_led
from umqtt.simple import MQTTClient
import network
import select
import struct
import ubinascii

# Definitions for pins
pin = Pin(28)
button = Pin(3, Pin.IN, Pin.PULL_DOWN)

# Definitions for NeoPixel LEDs
np = NeoPixel(pin, 30)
programming = 2          # Default to lamp

# Definitions for WiFi
SSID   = 'changeme'
PASSWD = 'changeme'
wlan   = network.WLAN(network.STA_IF)

# Definitions for MQTT Client
BROKER      = 'changeme'
BROKER_USER = 'changeme'
BROKER_PASS = 'changeme'
CLIENT_ID   = ubinascii.hexlify(machine.unique_id())
TOPIC       = b'led/#'
PUBTOPIC    = b'led/active' # Specific topic for switching on/off
T_PREFIX    = TOPIC.decode('utf-8')[:-1]
client      = MQTTClient(CLIENT_ID, BROKER, user=BROKER_USER, password=BROKER_PASS)

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

# Toggle on/off (through MQTT messaging)
# The state is subordinated to MQTT broker, so we need to broadcast the change
# and wait for it to come back
def toggle():
    global PUBTOPIC
    global client
    global blinking
    print('requesting status change...')
    client.publish(PUBTOPIC, str(int(not blinking)).encode(), retain=True, qos=0)
    print('requested')

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
        r, g, b = bytes.fromhex(colorstr)
        prog2_color = (r, g, b)
    except:
        pass

# Change current program
def set_program(programstr):
    global programming
    print(f'set program: {programstr}')
    try:
        prog = int(programstr)
        if (prog >= 0) and (prog < 3):
            programming = prog
            lights_off()
    except:
        pass

# Change on/off state
def set_blinking(value):
    global blinking
    print(f'set blinking: {value}')
    blinking = value
    if not blinking:
        lights_off()
        
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

# Dispatchers
def dispatch_active(value):
    set_blinking(value)

def dispatch_color(c):
    set_color(c)

def dispatch_dimmer(value):
    set_dim(value)

def dispatch_program(value):
    set_program(value)

# MQTT client callback
def mqtt_callback(t, m):
    global T_PREFIX
    topic    = t.decode('utf-8')
    message  = m.decode('utf-8')
    subtopic = topic[len(T_PREFIX):]
    try:
        if subtopic == 'dim':
            dispatch_dimmer(float(message))
        elif subtopic == 'program':
            dispatch_program(int(message))
        elif subtopic == 'color':
            dispatch_color(message)
        elif subtopic == 'active':
            dispatch_active(bool(int(message)))
        else:
            print('%s :: %s' % (topic, message))
    except:
        print('Error processing "%s :: %s"' % (topic, message))


# Start MQTT client
def start_client():
    global client
    global TOPIC
    global BROKER
    print(f'Connecting to MQTT Broker @ {BROKER}...')
    client.set_callback(mqtt_callback)
    client.connect()
    print('Connected. Subscribing to topics on range %s...' % TOPIC.decode('utf-8'))
    client.subscribe(TOPIC, qos=1)

# Poll MQTT client events
def poll_messages():
    global client
    client.check_msg()

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
        poll_messages()

if __name__ == "__main__":
    try:
        # Turn lights off if any
        lights_off()
    
        # Light on onboard LED
        pico_led.on()
    
        # Connect to WiFi
        ip = wlan_connect(SSID, PASSWD)

        # Start MQTT Client
        start_client()
    
        # Run on non-interpreter thread
        print('Starting main loop')
        blink_lights_loop()
    except:
        # On exceptions, restart RPi Pico W.
        # This avoids errors such as Address in Use for sockets.
        machine.reset()
