import asyncio
from bleak import BleakScanner

TARGET_MAC_ADDRESS="ED:67:39:0A:C5:C0"
FIXED_DATA="AABB ED67 390A C5C0"

def only_weight_data(raw_data):
    """
    Convert raw byte data into a human-readable hexadecimal format.
    """
    hex_data = raw_data.hex().upper()

    weight = int(hex_data[36:38] + hex_data[34:36], 16)/100.

    return str(weight) + "Kg"

def format_all_raw_data(raw_data):
    """
    Convert raw byte data into a human-readable hexadecimal format.
    """
    # Convert the byte data to a hex string
    hex_data = raw_data.hex().upper()

    # Format the hex string with spaces every 4 characters (byte pairs)
    formatted_data = ' '.join([hex_data[i:i+4] for i in range(0, len(hex_data), 4)])

    return formatted_data

async def scan_scale():
    """
    Scan for BLE devices and print formatted raw advertisement data from those sending manufacturer data.
    """
    print("Starting BLE scan...")
    scanner = BleakScanner()

    def detection_callback(device, advertisement_data):
        if device.address == TARGET_MAC_ADDRESS:
            for key, raw_data in advertisement_data.manufacturer_data.items():
                print(only_weight_data(raw_data))

    # Set the callback for when devices are discovered
    scanner.register_detection_callback(detection_callback)

    # Start scanning
    await scanner.start()
    try:
        # Run the scan for 30 seconds (or until manually stopped)
        await asyncio.sleep(240)
    finally:
        await scanner.stop()
        print("Scan complete.")

# Run the scanner
asyncio.run(scan_scale())
