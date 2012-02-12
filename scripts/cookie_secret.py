#!/usr/bin/env python
# coding: utf-8

import base64
import uuid

if __name__ == "__main__":
    print(base64.b64encode(uuid.uuid4().bytes + uuid.uuid4().bytes))
