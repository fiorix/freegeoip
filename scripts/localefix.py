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


import re
import sys

if __name__ == "__main__":
    try:
        filename = sys.argv[1]
        assert filename != "-"
        fd = open(filename)
    except:
        fd = sys.stdin

    line_re = re.compile(r'="([^"]+)"')
    for line in fd:
        line = line_re.sub(r"=\\1", line)
        sys.stdout.write(line)
    fd.close()
