![repo logo](docs/repo-logo.png)
![repo title](docs/repo-title.png)

---

[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=stephenlclarke_fixdecoder&metric=alert_status&token=693074ba90b11562241b1e602d8dc9ec0ef7bff5)](https://sonarcloud.io/summary/new_code?id=stephenlclarke_fixdecoder)
[![Bugs](https://sonarcloud.io/api/project_badges/measure?project=stephenlclarke_fixdecoder&metric=bugs&token=693074ba90b11562241b1e602d8dc9ec0ef7bff5)](https://sonarcloud.io/summary/new_code?id=stephenlclarke_fixdecoder)
[![Code Smells](https://sonarcloud.io/api/project_badges/measure?project=stephenlclarke_fixdecoder&metric=code_smells&token=693074ba90b11562241b1e602d8dc9ec0ef7bff5)](https://sonarcloud.io/summary/new_code?id=stephenlclarke_fixdecoder)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=stephenlclarke_fixdecoder&metric=coverage&token=693074ba90b11562241b1e602d8dc9ec0ef7bff5)](https://sonarcloud.io/summary/new_code?id=stephenlclarke_fixdecoder)
[![Duplicated Lines (%)](https://sonarcloud.io/api/project_badges/measure?project=stephenlclarke_fixdecoder&metric=duplicated_lines_density&token=693074ba90b11562241b1e602d8dc9ec0ef7bff5)](https://sonarcloud.io/summary/new_code?id=stephenlclarke_fixdecoder)
[![Lines of Code](https://sonarcloud.io/api/project_badges/measure?project=stephenlclarke_fixdecoder&metric=ncloc&token=693074ba90b11562241b1e602d8dc9ec0ef7bff5)](https://sonarcloud.io/summary/new_code?id=stephenlclarke_fixdecoder)
[![Reliability Rating](https://sonarcloud.io/api/project_badges/measure?project=stephenlclarke_fixdecoder&metric=reliability_rating&token=693074ba90b11562241b1e602d8dc9ec0ef7bff5)](https://sonarcloud.io/summary/new_code?id=stephenlclarke_fixdecoder)
[![Reliability Rating](https://sonarcloud.io/api/project_badges/measure?project=stephenlclarke_fixdecoder&metric=reliability_rating&token=693074ba90b11562241b1e602d8dc9ec0ef7bff5)](https://sonarcloud.io/summary/new_code?id=stephenlclarke_fixdecoder)
[![Technical Debt](https://sonarcloud.io/api/project_badges/measure?project=stephenlclarke_fixdecoder&metric=sqale_index&token=693074ba90b11562241b1e602d8dc9ec0ef7bff5)](https://sonarcloud.io/summary/new_code?id=stephenlclarke_fixdecoder)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=stephenlclarke_fixdecoder&metric=sqale_rating&token=693074ba90b11562241b1e602d8dc9ec0ef7bff5)](https://sonarcloud.io/summary/new_code?id=stephenlclarke_fixdecoder)
[![Vulnerabilities](https://sonarcloud.io/api/project_badges/measure?project=stephenlclarke_fixdecoder&metric=vulnerabilities&token=693074ba90b11562241b1e602d8dc9ec0ef7bff5)](https://sonarcloud.io/summary/new_code?id=stephenlclarke_fixdecoder)
---

# Steve's FIX Decoder / logfile prettify utility

This is my attempt to create an "all-singing / all-dancing" utility to pretty-print logfiles containing FIX Protocol messages while simultaneously learning golang and trying to incorporate SonarQube Code Quality metrics.

I have written utilities like this in past in Java, Python, C, C++ and even in Bash/Awk!! This is my favourite one so far. Maybe Rust will be next.

![repo title](docs/example.png)

# How to use it

The utility behaves like the `cat` utility in `Linux`, except as it reads the input (either piped in from `stdin` or from a filename specified on the commandline) it scans each line for `FIX protocol` messages and prints them out highlighted in bold white while the rest of the line will be in a mid grey colour. After the line is output it will be followed by a detailed breakdown of all the `FIX Protocol` tags that were found in the message. The detailed output will use the appropriate `FIX` dictionary for the version of `FIX` specified in `BeginString (tag 8)` tag.

I plan to produce an update shortly that will also look at `DefaultApplVerID (tag 1137)` when `8=FIXT.1.1` is detected in the message.

## Running the utility

```bash
❯ bin/fixdecoder-2.0.3-develop.darwin-arm64 --help
fixdecoder v2.0.3-develop (branch:develop, commit:01dca64)
  git clone git@github.com:stephenlclarke/fixdecoder.git
Usage: fixdecoder [[--fix=44] | [--xml=FIX44.xml]] [--message[=MSG] [--verbose] [--column] [--header] [--trailer]]
       fixdecoder [[--fix=44] | [--xml=FIX44.xml]] [--tag[=TAG] [--verbose] [--column]]
       fixdecoder [[--fix=44] | [--xml=FIX44.xml]] [--component=[NAME] [--verbose]]
       fixdecoder [[--fix=44] | [--xml=FIX44.xml]] [--info]
       fixdecoder [--validate] [--colour=yes|no] [--secret] [file1.log file2.log ...]
       fixdecoder [--version]

Flags:
  -colour
      Force coloured output (yes|no). Default: auto-detect based on stdout
  -column
      Display enums in columns
  -component
      Component to display (omit to list all components)
  -fix string
      FIX version to use (40,41,42,43,44,50,50SP1,50SP2,T11) (default "44")
  -header
      Include Header block
  -info
      Show XML schema summary (fields, components, messages, version counts)
  -message
      Message name or MsgType (omit to list all messages)
  -secret
      Obfuscate sensitive FIX tag values
  -tag
      Tag number to display details for (omit to list all tags)
  -trailer
      Include Trailer block
  -validate
      Validate FIX messages during decoding
  -verbose
      Show full message structure with enums
  -version
      Print version information and exit
  -xml string
      Path to alternative FIX XML file

❯ ./bin/fixdecoder/v2.0.3-develop/fixdecoder --help
fixdecoder v2.0.3-develop (branch:develop, commit:f3c0f91)

  git clone git@github.com:stephenlclarke/fixdecoder.git

Usage: fixdecoder [[--fix=44] | [--xml FIX44.xml]] [--message[=MSG] [--verbose] [--column] [--header] [--trailer]]
       fixdecoder [[--fix=44] | [--xml FIX44.xml]] [--tag[=TAG] [--verbose] [--column]]
       fixdecoder [[--fix=44] | [--xml FIX44.xml]] [--component=[NAME] [--verbose]]
       fixdecoder [[--fix=44] | [--xml FIX44.xml]] [--info]
       fixdecoder [--validate] [--colour=yes|no] [file1.log file2.log ...]

Flags:
  --colour
        Force coloured output (yes|no). Default: auto-detect based on stdout
  --column
        Display enums in columns
  --component
        Component to display (omit to list all components)
  --fix string
        FIX version to use (40,41,42,43,44,50,50SP1,50SP2,T11) (default "44")
  --header
        Include Header block
  --info
        Show XML schema summary (fields, components, messages, version counts)
  --message
        Message name or MsgType (omit to list all messages)
  --secret
        Obfuscate sensitive FIX tag values
  --tag
        Tag number to display details for (omit to list all tags)
  --trailer
        Include Trailer block
  --validate
        Validate FIX messages during decoding
  --verbose
        Show full message structure with enums
  --xml string
        Path to alternative FIX XML file
```

## How to get it

ℹ️ However you download it you will have to make the binary executable on your
computer. **Windows** users will need to rename the download and add a `.exe`
extension to the binary before you can execute it. **Linux** and **MacOS**
users will need to do a `chmod +x` on the file first.

### Download it BITBUCKET-ONLY

Check out the Repo's [Download Page](https://github.com/stephenlclarke/fixdecoder/downloads/) to see what versions are available for the computer you want to run it on.

![repo logo](docs/repo-download.png) BITBUCKET-ONLY

Or by downloading the artifacts from the S3 bucket;

```bash
❯ aws s3 ls s3://stephenlclarke/release/fixdecoder/ --recursive --human-readable --summarize
2025-07-22 19:55:19    7.2 MiB release/fixdecoder/v2.0.2/fixdecoder-2.0.2.darwin-arm64
2025-07-22 19:55:19    7.3 MiB release/fixdecoder/v2.0.2/fixdecoder-2.0.2.linux-amd64
2025-07-22 19:55:19    7.3 MiB release/fixdecoder/v2.0.2/fixdecoder-2.0.2.linux-arm64
2025-07-22 19:55:19    7.6 MiB release/fixdecoder/v2.0.2/fixdecoder-2.0.2.windows-amd64
2025-07-22 20:15:25    7.2 MiB release/fixdecoder/v2.0.3/fixdecoder-2.0.3.darwin-arm64
2025-07-22 20:15:25    7.3 MiB release/fixdecoder/v2.0.3/fixdecoder-2.0.3.linux-amd64
2025-07-22 20:15:25    7.3 MiB release/fixdecoder/v2.0.3/fixdecoder-2.0.3.linux-arm64
2025-07-22 20:15:25    7.6 MiB release/fixdecoder/v2.0.3/fixdecoder-2.0.3.windows-amd64

Total Objects: 8
   Total Size: 58.7 MiB
```

Choose the file that matches the machine operating system and architecture that you want and download it from the s3 bucket.

```bash
❯ aws s3 cp s3://stephenlclarke/release/fixdecoder/v2.0.3-develop/fixdecoder-2.0.3-develop.darwin-arm64 ./fixdecoder
download: s3://stephenlclarke/release/fixdecoder/v2.0.3-develop/fixdecoder-2.0.3-develop.darwin-arm64 to ./fixdecoder
❯ chmod +x ./fixdecoder
❯ ./fixdecoder --version
fixdecoder v2.0.3-develop (branch:develop, commit:c2a60e8)
  git clone git@github.com:stephenlclarke/fixdecoder.git
```

### Build it

Build it from source. This require `bash` version 5+ and `go` version 1.25.0

```bash
❯ bash --version
GNU bash, version 5.3.3(1)-release (aarch64-apple-darwin24.4.0)
Copyright (C) 2025 Free Software Foundation, Inc.
License GPLv3+: GNU GPL version 3 or later <http://gnu.org/licenses/gpl.html>

This is free software; you are free to change and redistribute it.
There is NO WARRANTY, to the extent permitted by law.

❯ go version
go version go1.25.0 darwin/arm64
```

Clone the git repo.

```bash
❯ git clone git@github.com:stephenlclarke/fixdecoder.git
Cloning into 'fixdecoder'...
remote: Enumerating objects: 418, done.
remote: Counting objects: 100% (418/418), done.
remote: Compressing objects: 100% (375/375), done.
remote: Total 418 (delta 201), reused 0 (delta 0), pack-reused 0 (from 0)
Receiving objects: 100% (418/418), 1.02 MiB | 2.65 MiB/s, done.
Resolving deltas: 100% (201/201), done.
❯ cd fixdecoder
```

Then build it.

```bash
❯ ./ci.sh build

>> Setting up environment

>> Installing test dependencies

>> Running go mod tidy in all modules

>> Auto-Generating FIX dictionary
Downloading https://raw.githubusercontent.com/quickfix/quickfix/master/spec/FIX40.xml → resources/FIX40.xml
Downloading https://raw.githubusercontent.com/quickfix/quickfix/master/spec/FIX41.xml → resources/FIX41.xml
Downloading https://raw.githubusercontent.com/quickfix/quickfix/master/spec/FIX42.xml → resources/FIX42.xml
Downloading https://raw.githubusercontent.com/quickfix/quickfix/master/spec/FIX43.xml → resources/FIX43.xml
Downloading https://raw.githubusercontent.com/quickfix/quickfix/master/spec/FIX44.xml → resources/FIX44.xml
Downloading https://raw.githubusercontent.com/quickfix/quickfix/master/spec/FIX50.xml → resources/FIX50.xml
Downloading https://raw.githubusercontent.com/quickfix/quickfix/master/spec/FIX50SP1.xml → resources/FIX50SP1.xml
Downloading https://raw.githubusercontent.com/quickfix/quickfix/master/spec/FIX50SP2.xml → resources/FIX50SP2.xml
Downloading https://raw.githubusercontent.com/quickfix/quickfix/master/spec/FIXT11.xml → resources/FIXT11.xml
Processing resources/FIX40.xml → fix/fix40/fix40.go
Processing resources/FIX41.xml → fix/fix41/fix41.go
Processing resources/FIX42.xml → fix/fix42/fix42.go
Processing resources/FIX43.xml → fix/fix43/fix43.go
Processing resources/FIX44.xml → fix/fix44/fix44.go
Processing resources/FIX50.xml → fix/fix50/fix50.go
Processing resources/FIX50SP1.xml → fix/fix50SP1/fix50SP1.go
Processing resources/FIX50SP2.xml → fix/fix50SP2/fix50SP2.go
Processing resources/FIXT11.xml → fix/fixT11/fixT11.go
Generating fix/chooseFixVersion.go
Done.

>> Auto-Generating FIX sensitive tags
Generated fix/sensitiveTagNames.go with 109 tags

>> Building fixdecoder v2.0.3-develop (branch: develop, commit: 570547b), OS: darwin, ARCH: arm64

real  0m0.392s
user  0m0.221s
sys   0m0.228s

>> Copying binaries
[Sep 16 21:28]  ./bin
├── [Sep 16 21:28]  fixdecoder
└── [Sep 16 21:28]  fixdecoder-2.0.3-develop
    └── [Sep 16 21:28]  fixdecoder

2 directories, 2 files
```

Run it and check the version details

```bash
❯ ./bin/fixdecoder/v2.0.3-develop/fixdecoder --version
fixdecoder v2.0.3-develop (branch:develop, commit:c2a60e8)
  git clone git@github.com:stephenlclarke/fixdecoder.git
```
