#!/usr/bin/perl

use strict;
use warnings;

use Data::Dumper;

my $struct = [
	'config', '*common.Configuration', { panic => 1 },
	'logger', '*logger.Logger', { panic => 1 },

	'metadataBackend', '*metadata.Backend', { panic => 1 },
	'dataBackend', 'data.Backend', { panic => 1 },
	'streamBackend', 'data.Backend', { panic => 1 },
	'authenticator', '*common.SessionAuthenticator', { panic => 1 },

    'pagingQuery',  '*common.PagingQuery', { panic => 1 },

	'sourceIP', 'net.IP', {},

	'upload', '*common.Upload', {},
	'file', '*common.File', {},
	'user', '*common.User', {},
	'token', '*common.Token', {},

	'isWhitelisted', '*bool', { internal => 1 },
	'isRedirectOnFailure', 'bool', {},
	'isQuick', 'bool', {},

	'req', '*http.Request', {},
	'resp', 'http.ResponseWriter', {},

	'mu', 'sync.RWMutex', { internal => 1 },
];

sub genGet
{
    my $param = shift;
    my $type = shift;
    my $params = shift;

    return "" if $params->{'internal'};

    my $uc = ucfirst $param;

    my $str = "";
    if ( $type eq 'bool' ) {
        $str = << "EOF";
// $uc get $param from the context.
func (ctx *Context) $uc() $type {
    ctx.mu.RLock()
    defer ctx.mu.RUnlock()

    return ctx.$param
}
EOF
    } else {
        $str = << "EOF";
// Get$uc get $param from the context.
func (ctx *Context) Get$uc() $type {
    ctx.mu.RLock()
    defer ctx.mu.RUnlock()

EOF
        if ( $params->{'panic'} ) {
        $str .= << "EOF";
    if ctx.$param == nil {
        panic("missing $param from context")
    }

EOF
        }

    $str .= << "EOF";
    return ctx.$param
}
EOF
    }
}

sub genSet
{
    my $param = shift;
    my $type = shift;
    my $params = shift;

    return "" if $params->{'internal'};

    my $uc = ucfirst $param;

    if ( $type eq 'bool' ) {
        $uc =~ s/^Is//
    }

    my $str = << "EOF";
// Set$uc set $param in the context
func (ctx *Context) Set$uc($param $type) {
    ctx.mu.Lock()
    defer ctx.mu.Unlock()

    ctx.$param = $param
}

EOF

    return $str;
}

sub genImports
{
    my $str = << 'EOF';
package context

import (
	"net"
	"net/http"
	"sync"

	"github.com/root-gg/logger"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/data"
	"github.com/root-gg/plik/server/metadata"
)

EOF
    return $str;
}

sub genStruct
{
    my $struct = shift;

    my $str = "// Context to be propagated throughout the middleware chain\n";
    $str .= "type Context struct {\n";
    for (my $i = 0 ; $i < @$struct ; $i += 3)
    {
        my $param = $struct->[$i];
        my $type = $struct->[$i + 1];

        $str .= "\t$param $type\n";
    }
    $str .= "}\n";

    return $str;
}

sub genMethods
{
    my $struct = shift;

    my $str = "";
    for (my $i = 0 ; $i < @$struct ; $i += 3)
    {
        my $param = $struct->[$i];
        my $type = $struct->[$i + 1];
        my $params = $struct->[$i + 2];

        $str .= genGet($param, $type, $params);
        $str .= genSet($param, $type, $params);
    }

    return $str;
}

sub genCode
{
    my $struct = shift;

    my $str = genImports;
    $str .= "\n";
    $str .= genStruct $struct;
    $str .= "\n";
    $str .= genMethods $struct;
    $str .= "\n";
}

print genCode $struct;