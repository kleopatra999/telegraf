# Telegraf Data Formats

There are many Telegraf plugins that are able to parse generic text data from
a variety of sources, this includes parsing the stdout of executed scripts
(`exec`) and parsing messages received from message brokers (`kafka_consumer`).

Up until now, these plugins were statically configured to parse just a single
data format. `exec` mostly only supported parsing JSON, and `kafka_consumer` only
supported data in InfluxDB line-protocol.

But now we are normalizing the parsing of various data formats across all
plugins that can support it. You will be able to tell a plugin that supports
different data formats by the presence of a `data_format` config option, for
example, in the exec plugin:

```
[[inputs.exec]]
  ### Commands array
  commands = ["/tmp/test.sh", "/usr/bin/mycollector --foo=bar"]

  ### measurement name suffix (for separating different commands)
  name_suffix = "_mycollector"

  ### Data format to consume. This can be "json", "influx" or "graphite" (line-protocol)
  ### Each data format has it's own unique set of configuration options, read
  ### more about them here:
  ### https://github.com/influxdata/telegraf/blob/master/DATA_FORMATS.md
  data_format = "json"
```

## Influx line-protocol Options:

None!

## JSON Options:

```
### List of tag names to extract from top-level of JSON server response
tag_keys = [
  "my_tag_1",
  "my_tag_2"
]
```

## Graphite Options:

```
### Below configuration will be used for data_format = "graphite", can be ignored for other data_format
### If matching multiple measurement files, this string will be used to join the matched values.
separator = "."

### Each template line requires a template pattern.  It can have an optional
### filter before the template and separated by spaces.  It can also have optional extra
### tags following the template.  Multiple tags should be separated by commas and no spaces
### similar to the line protocol format.  The can be only one default template.
### Templates support below format:
### 1. filter + template
### 2. filter + template + extra tag
### 3. filter + template with field key
### 4. default template
templates = [
  "*.app env.service.resource.measurement",
  "stats.* .host.measurement* region=us-west,agent=sensu",
  "stats2.* .host.measurement.field",
  "measurement*"
]
```
