# Renpho ES CS20M

## Why?

I weight myself every day, when I wake up, and I keep track of my weight in a Google Sheet.
What often happens is that, in the short way between the scale and the toilet,
I forget to jot the weight down, and then I forget the weight itself.

Thanks to this, I just need to weight myself, and the Google Sheet will automatically get updated.

## How?

This is composed of two parts:
- The Weight Updater adds a weight to the Google Sheet in which I track my weight
  - It is deployed as a GCP Cloud Run Function, that takes a weight/date combination as an input, and writes it in the appropriate place in the Google Sheet
- The Weight Receiver scans for BLE messages coming from my scale, and when it spots a stable weight, it transmits it to the Weight Updater
  - This runs as a docker container on a Raspberry Pi 2, conveniently located to receive messages from my scale.

### Why not write in the Google Sheet directly from the Raspberry Pi?

I felt like trying GCP Cloud Run Functions. This way, the Raspberry Pi is exclusively authorised to call a certain function, and not to touch the Google Sheet directly.