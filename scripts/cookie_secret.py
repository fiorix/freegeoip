#!/usr/bin/env python
# coding: utf-8
#
# Copyright 2013 Alexandre Fiori
# Powered by cyclone
#
# Licensed under the Apache License, Version 2.0 (the "License"); you may
# not use this file except in compliance with the License. You may obtain
# a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
# WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
# License for the specific language governing permissions and limitations
# under the License.


import base64
import uuid

if __name__ == "__main__":
    print(base64.b64encode(uuid.uuid4().bytes + uuid.uuid4().bytes))
