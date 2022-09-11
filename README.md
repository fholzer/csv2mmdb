# CSV to Maxmind mmdb Format Converter

This is a utility for converting CSV files to MaxMind's mmdb format.

# Usage

Required arguments:

* `-inpupt=[FILENAME]` - Path to the CSV input file.
* `-output=[FILENAME]` - Path to the mmdb output file
* `-config=[FILENAME]` - Path to the configuration file

# Development
Here are some usefull resources:
* Look up DB formats here: https://github.com/runk/mmdb-lib/blob/master/src/reader/response.ts
* mmdb format spec: https://maxmind.github.io/MaxMind-DB/index.html
* response spec: https://dev.maxmind.com/geoip/docs/web-services/responses?lang=en#object-reference
# Copyright and License

This software is Copyright (c) 2022 by Ferdinand Holzer

Files in the pkg/convert/internal/valuecache directory are derivative
work of https://github.com/maxmind/mmdbwriter, (c) by MaxMind, Inc. They
are redistributed under the MIT license. Find a copy of the MIT license
at pkg/convert/internal/valuecache/LICENSE-MIT

The remaining project is licensed under the GNU GENERAL PUBLIC LICENSE
Version 3. Find a copy of the license in the LICENSE file.