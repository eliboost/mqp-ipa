#!/bin/bash

CONFIG="/var/lib/artemis/etc/broker.xml"

if grep -q "<address-settings>" "$CONFIG"; then
  sed -i '/<address-settings>/a \
    <address-setting match="#"> \
      <auto-create-queues>true</auto-create-queues> \
      <auto-create-addresses>true</auto-create-addresses> \
    </address-setting>' "$CONFIG"
else
  echo "ERROR: No <address-settings> section found."
fi
