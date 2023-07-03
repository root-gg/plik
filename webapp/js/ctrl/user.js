// HTTP basic auth credentials dialog controller
plik.controller('UserController', ['$scope', 'args', '$config', '$q',
    function ($scope, args, $config, $q) {
        $scope.title = 'User :';

        $scope.providers = ["local", "google", "ovh"];
        $scope.edit = false;
        $scope.user = {};
        $scope.warning = null;

        $scope.configReady = $q.defer();
        $config.getConfig()
            .then(function (config) {
                $scope.config = config;
                $scope.configReady.resolve(true);
            }).then(null, function (error) {
            $scope.$close({error: error});
        });

        $scope.userReady = $q.defer();
        $config.getUser()
            .then(function (user) {
                $scope.auth_user = user;
                $scope.userReady.resolve(true);
            }).then(null, function (error) {
            $scope.$close({error: error});
        });


        $scope.maxFileSize = -1;
        $scope.ttlUnits = ttlUnits;
        $scope.ttlUnits[3] = "unlimited";
        $scope.ttlUnit = "days";
        $scope.ttlValue = 30;

        // Set MaxTTL value
        $scope.setMaxTTL = function (ttl) {
            var res = getHumanReadableTTL(ttl)
            $scope.ttlValue = res[0]
            $scope.ttlUnit = $scope.ttlUnits[res[2]];
        };

        // Set MaxFileSize value
        $scope.setMaxFileSize = function (maxFileSize) {
            if (maxFileSize > 0) {
                $scope.maxFileSize = getHumanReadableSize(maxFileSize);
            } else {
                $scope.maxFileSize = maxFileSize;
            }
        }

        // Set MaxUserSize value
        $scope.setMaxUserSize = function (maxUserSize) {
            if (maxUserSize > 0) {
                $scope.maxUserSize = getHumanReadableSize(maxUserSize);
            } else {
                $scope.maxUserSize = maxUserSize;
            }
        }
        
        // whenReady ensure that the scope has been initialized especially :
        // $scope.config, $scope.user, $scope.mode, $scope.upload, $scope.files, ...
        $scope.ready = $q.all([$scope.configReady, $scope.userReady]);

        $scope.ready
            .then(function () {
                if (args.user) {
                    // Paranoid useless check
                    if (!$scope.auth_user.admin && args.user.id !== $scope.auth_user.id) {
                        $scope.closeWithError("forbidden")
                        return;
                    }

                    $scope.edit = true;
                    $scope.user = args.user;
                    $scope.setMaxTTL($scope.user.maxTTL);
                    $scope.setMaxFileSize($scope.user.maxFileSize);
                    $scope.setMaxUserSize($scope.user.maxUserSize);
                } else {
                    $scope.user.provider = "local";
                    $scope.setMaxTTL(0);
                    $scope.setMaxFileSize(0);
                    $scope.setMaxUserSize(0)
                    $scope.generatePassword();
                }
            }).then(function () {
            // discard
        })

        // Generate random 16 chars
        $scope.generatePassword = function () {
            pass = "";
            for (i=0;i<2;i++) {
                pass += window.crypto.getRandomValues(new BigUint64Array(1))[0].toString(36)
            }
            $scope.user.password = pass;
        }

        // Check TTL value
        $scope.checkTTL = function (ttl) {
            // Invalid negative value
            if ($scope.ttlUnit !== 'unlimited' && ttl < 0) {
                $scope.warning = "Invalid max TTL : " + getHumanReadableTTLString(ttl);
                return false;
            }

            return true;
        };

        $scope.check = function(user) {
            $scope.warning = null;

            if (!$scope.edit && (!user.login || user.login.length < 4)) {
                $scope.warning = "invalid login (min 4 chars)";
                return false;
            }

            if (!($scope.edit && !user.password)) {
                if (!user.password || user.password.length < 8) {
                    $scope.warning = "invalid password (min 8 chars)";
                    return false;
                }
            }

            // Get TTL in seconds
            var ttl = getTTL($scope.ttlValue, $scope.ttlUnit);
            if (!$scope.checkTTL(ttl)) {
                return false;
            }
            $scope.user.maxTTL = ttl;

            // Parse maxFileSize
            var maxFileSize = parseHumanReadableSize($scope.maxFileSize, {base: 10});
            if (_.isNumber(maxFileSize)) {
                $scope.user.maxFileSize = maxFileSize;
            } else {
                maxFileSize = Number($scope.maxFileSize)
                if (maxFileSize === 0 || maxFileSize === -1) {
                    $scope.user.maxFileSize = maxFileSize;
                } else {
                    $scope.warning = "invalid max file size";
                    return false;
                }
            }

            // Parse maxFileSize
            var maxUserSize = parseHumanReadableSize($scope.maxUserSize, {base: 10});
            if (_.isNumber(maxUserSize)) {
                $scope.user.maxUserSize = maxUserSize;
            } else {
                maxUserSize = Number($scope.maxUserSize)
                if (maxUserSize === 0 || maxUserSize === -1) {
                    $scope.user.maxUserSize = maxUserSize;
                } else {
                    $scope.warning = "invalid max user size";
                    return false;
                }
            }

            return true;
        };

        $scope.closeWithError = function (error) {
            $scope.$close({error: error});
        }

        $scope.close = function (user) {
            if ($scope.check(user)) {
                $scope.$close({user: user});
            }
        };
    }]);