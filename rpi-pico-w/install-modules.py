import network
import mip

wlan = network.WLAN(network.STA_IF)
wlan.active(True)
wlan.scan()
wlan.connect('changeme', 'changeme')
while not wlan.isconnected():
    pass
ip = wlan.ifconfig()[0]


mip.install("neopixel")
mip.install("https://raw.githubusercontent.com/RaspberryPiFoundation/picozero/master/picozero/picozero.py")
mip.install("umqtt.simple")


