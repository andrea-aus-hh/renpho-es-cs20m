# 

## What and Why?

Every day, when I wake up, I weight myself on my Renpho ES CS20M scale, I go back to the bedroom and note the weight in a Google Sheet.
What often happens is that I'll forget it in the few steps between the scale and the bedroom.

Thanks to this system, I just need to weight myself, and the Google Sheet automatically gets updated.

## How?

This is composed of two parts:
- The **Weight Updater** is responsible for writing the incoming weight to the Google Sheet.
  - It is deployed as a GCP Cloud Run Function which takes a weight/date as an input, and writes it in the appropriate place in the Google Sheet.
  - Why a Google Sheet? I work with a personal trainer, and our main way of keeping track of data is Google Sheets.
- The **Weight Scanner** listens for BLE messages coming from my scale, parses the weight data, and sends it to the Weight Updater.
  - This runs as a service on a Raspberry Pi 1, conveniently located to receive messages from my scale.

### Why not write in the Google Sheet directly from the Raspberry Pi?

It would definitively be possible, and it would be a simpler design. This was for me an exercise in Terraform, GCP and GCP Cloud Run Functions.
Also, this way the Raspberry Pi has fewer responsibilities, it only needs to call an API endpoint.

### How is the weight parsed from the scale's payload?

This is how a sequence of BLE messages coming from my scale looks like:

```
AABB ED67 390A C5C0 332F 4167 FFFF FF02 3000 004B 0903 B012  (Zero weight)
AABB ED67 390A C5C0 332F 4167 FFFF FF02 30A3 024B 0903 B112  (Weight applied, value starts to change)
AABB ED67 390A C5C0 342F 4167 FFFF FF02 304C 1D4B 0903 B212  (Stable weight)
AABB ED67 390A C5C0 352F 4167 FFFF FF02 309D 034B 0903 B312  (Further fluctuations)
AABB ED67 390A C5C0 352F 4167 FFFF FF02 3000 004B 0903 B412  (Back to zero weight)
```

Such messages can be divided in the following parts:
-  `AABB ED67 390A C5C0 352F 4167 FFFF FF02 30` never changes
- The next four characters (2 bytes) are the measured weight, in little-endian order
- `4B` is fixed, and represents the letter `K` in ASCII, likely referring to Kilograms
- The rest appears to be further measurements, which I didn't bother decoding
  - Consider that the scale also measures BMI and other values, so it might be that data

The filtering of messages is based on the MAC Address of the scale, which I found
by trying to isolate myself as much as possible from other BLE devices,
and finding the one that was sending messages only when stepping on the scale.
