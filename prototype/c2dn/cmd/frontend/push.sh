#!/usr/bin/env bash
curl -s -i -o samplePushRequest2 "http://asrock.jasony.me:8080/akamai/abcd_25"
curl -x -s -o /dev/null -X PUSH --data-binary samplePushRequest "http://asrock.jasony.me:8080/push.html"
curl -vvv -x -X PUSH --data-binary samplePushRequest2 "http://asrock.jasony.me:8080/push.html"






#!/usr/bin/env perl

#
# Licensed to the Apache Software Foundation (ASF) under one
# or more contributor license agreements.  See the NOTICE file
# distributed with this work for additional information
# regarding copyright ownership.  The ASF licenses this file
# to you under the Apache License, Version 2.0 (the
# "License"); you may not use this file except in compliance
# with the License.  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

use strict;
use warnings;

use bytes;
use IO::Socket qw( :crlf );
use Getopt::Long;
use URI;


die ("Use --help for options\n") unless GetOptions (
	"f|file=s" => \my $f_name,
        "u|url=s" => \my $url,
	"v|verbose" => \my $debug,
        "h|help" => sub {
		print <<HELP;
Usage: $0 OPTIONS

OPTIONS

	-f|--file     Which file to PUSH
	-u|--url      URL where to PUSH
	-v|--verbose  Print the PUSHed content to stdout

Example
	$0 -f foo.txt -u http://localhost:8080/foo.txt
HELP
		exit;
	}
);

die ("--file and --url must be given!" ) unless ( $f_name && $url) ;

open (my $fh, '<', $f_name) or die $!;
my $uri = URI->new($url);
my $f_type = `file -b --mime $f_name`;

#
# read the file in one go:
#
binmode $fh;
my $f = do { local $/; <$fh> };
my $len_content = length($f) + 2;


my $response = "HTTP/1.0 200 OK${CRLF}Content-type: ${f_type}${CRLF}Content-length: ${len_content}${CRLF}${CRLF}${f}${CRLF}";
my $len_push = length $response;

my $push = "PUSH ${url} HTTP/1.0${CRLF}Content-Length: ${len_push}${CRLF}${CRLF}${response}";
print $push if ($debug);

my $sock = IO::Socket::INET->new(PeerAddr => $uri->host, PeerPort => $uri->port, Proto => 'tcp') or die "Error creating socket: $!";
print $sock $push;
print do { local $/; <$sock> };
