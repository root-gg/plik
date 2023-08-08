ttlUnits = ["days", "hours", "minutes", "unlimited"];

// Return TTL unit and value
function getHumanReadableTTL(ttl) {
    var value, unit, idx;
    if (ttl === -1) {
        value = -1;
        unit = "unlimited"
        idx = 3
    } else if (ttl < 3600) {
        value = Math.round(ttl / 60);
        unit = "minutes"
        idx = 2
    } else if (ttl < 86400) {
        value = Math.round(ttl / 3600);
        unit = "hours"
        idx = 1
    } else if (ttl >= 86400) {
        value = Math.round(ttl / 86400);
        unit = "days"
        idx = 0
    } else {
        value = 0;
        unit = "invalid";
        idx = 0
    }
    return [value, unit, idx];
}

// Return TTL as a string
function getHumanReadableTTLString(ttl) {
    var res = getHumanReadableTTL(ttl)
    if (res[0] > 0) {
        return res[0] + " " + res[1];
    }
    return res[1];
}

// Return TTL value in seconds
function getTTL (ttl, unit) {
    ttl = Number(ttl)
    if (unit === "minutes") {
        return ttl * 60;
    } else if (unit === "hours") {
        return ttl * 3600;
    } else if (unit === "days") {
        return ttl * 86400;
    } else {
        return -1;
    }
}

// Return human-readable filesize
function getHumanReadableSize(size) {
    if (_.isUndefined(size)) return;
    if (size === -1) return "unlimited"
    return filesize(size, {base: 10});
}

var validAmount  = function(n) {
    return !isNaN(parseFloat(n)) && isFinite(n);
};

var parsableUnit = function(u) {
    return u.match(/\D*/).pop() === u;
};

var incrementBases = {
    2: [
        [["b", "bit", "bits"], 1/8],
        [["B", "Byte", "Bytes", "bytes"], 1],
        [["Kb"], 128],
        [["k", "K", "kb", "KB", "KiB", "Ki", "ki"], 1024],
        [["Mb"], 131072],
        [["m", "M", "mb", "MB", "MiB", "Mi", "mi"], Math.pow(1024, 2)],
        [["Gb"], 1.342e+8],
        [["g", "G", "gb", "GB", "GiB", "Gi", "gi"], Math.pow(1024, 3)],
        [["Tb"], 1.374e+11],
        [["t", "T", "tb", "TB", "TiB", "Ti", "ti"], Math.pow(1024, 4)],
        [["Pb"], 1.407e+14],
        [["p", "P", "pb", "PB", "PiB", "Pi", "pi"], Math.pow(1024, 5)],
        [["Eb"], 1.441e+17],
        [["e", "E", "eb", "EB", "EiB", "Ei", "ei"], Math.pow(1024, 6)]
    ],
    10: [
        [["b", "bit", "bits"], 1/8],
        [["B", "Byte", "Bytes", "bytes"], 1],
        [["Kb"], 125],
        [["k", "K", "kb", "KB", "KiB", "Ki", "ki"], 1000],
        [["Mb"], 125000],
        [["m", "M", "mb", "MB", "MiB", "Mi", "mi"], 1.0e+6],
        [["Gb"], 1.25e+8],
        [["g", "G", "gb", "GB", "GiB", "Gi", "gi"], 1.0e+9],
        [["Tb"], 1.25e+11],
        [["t", "T", "tb", "TB", "TiB", "Ti", "ti"], 1.0e+12],
        [["Pb"], 1.25e+14],
        [["p", "P", "pb", "PB", "PiB", "Pi", "pi"], 1.0e+15],
        [["Eb"], 1.25e+17],
        [["e", "E", "eb", "EB", "EiB", "Ei", "ei"], 1.0e+18]
    ]
};

// from https://github.com/patrickkettner/filesize-parser/blob/master/index.js
function parseHumanReadableSize(input) {
    if (_.isUndefined(input)) return;
    var options = arguments[1] || {};
    var base = parseInt(options.base || 2);

    var parsed = input.toString().match(/^([0-9\.,]*)(?:\s*)?(.*)$/);
    var amount = parsed[1].replace(',','.');
    var unit = parsed[2];

    var validUnit = function(sourceUnit) {
        return sourceUnit === unit;
    };

    if (!validAmount(amount) || !parsableUnit(unit)) {
        return;
    }
    if (unit === '') return Math.round(Number(amount));

    var increments = incrementBases[base];
    for (var i = 0; i < increments.length; i++) {
        var _increment = increments[i];

        if (_increment[0].some(validUnit)) {
            return Math.round(amount * _increment[1]);
        }
    }
}