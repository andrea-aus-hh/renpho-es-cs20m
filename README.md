# Renpho ES CS20M

## What and Why?

Every day, when I wake up, I weight myself, I go back to the bedroom and write the weight in a Google Sheet.
What often happens is that, in the short way between the scale and the bedroom,
I forget to jot down the value.

Thanks to the system you'll find in this repository,
I just need to weight myself, and the Google Sheet automatically gets updated.

## How?

This is composed of two parts:
- The **Weight Updater** is responsible for writing the incoming weight to the Google Sheet
  - It is deployed as a GCP Cloud Run Function, that takes a weight/date combination as an input, and writes it in the appropriate place in the Google Sheet
  - Why a Google Sheet? Because it's shared with my personal trainer.
- The **Weight Scanner** listens for BLE messages, filters the ones from my scale, parses the weight, and sends it to the Weight Updater
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
-  `AABB ED67 390A C5C0 352F 4167 FFFF FF02 30` is fixed
- The next four characters (2 bytes) are the measured weight, in little-endian order
- The rest still appears to be body-related data, which I didn't bother decoding
  - Consider that the scale also measures BMI and other values, so it might be that data

I found the MAC Address of the scale by trying to isolate myself as much as possible from other BLE devices,
and finding one that was sending messages only when stepping on the scale.
