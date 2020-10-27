// Register controller
plik.controller('RegisterCtrl', ['$scope', '$api', '$config', '$location', '$dialog',
    function ($scope, $api, $config, $location, $dialog) {

        // Ugly but it works
        setTimeout(function () {
            $("#login").focus();
        }, 100);

        // Initialize register controller
        $scope.init = function () {
            $scope.invite = $location.search().invite;
            $scope.email = $location.search().email;
            if ($scope.email) {
                $scope.email_read_only = true;
            }
            //$location.search({});
        };

        // Get server config
        $config.getConfig()
            .then(function (config) {
                $scope.config = config;
                // Check if token authentication is enabled server side
                if (!config.authentication || config.registration === 'closed') {
                    $location.path('/');
                }
            })
            .then(null, function (error) {
                $dialog.alert(error);
            });

        // Get user from session
        $config.getUser()
            .then(function (user) {
                if (user.verified) {
                    $location.path('/home');
                } else {
                    $location.path('/confirm');
                }
            })
            .then(null, function (error) {
                if (error.status !== 401 && error.status !== 403) {
                    $dialog.alert(error);
                }
            });

        // Google authentication
        $scope.google = function () {
            $api.login("google", null, null, $scope.invite)
                .then(function (url) {
                    // Redirect to Google user consent dialog
                    window.location.replace(url);
                })
                .then(null, function (error) {
                    $dialog.alert(error);
                });
        };

        // OVH authentication
        $scope.ovh = function () {
            $api.login("ovh", null, null, $scope.invite)
                .then(function (url) {
                    // Redirect to OVH user consent dialog
                    window.location.replace(url);
                })
                .then(null, function (error) {
                    $dialog.alert(error);
                });
        };

        // Login with local user
        $scope.signup = function () {
            if ($scope.password !== $scope.confirmPassword) {
                $dialog.alert("password confirmation mismatch");
                return
            }

            var params = {
                login: $scope.login,
                name: $scope.name,
                email: $scope.email,
                password: $scope.password,
            };

            if ($scope.invite) {
                params.invite = $scope.invite;
            }

            $api.register(params)
                .then(function (user) {
                    $config.refreshUser();
                    if (user.verified) {
                        $location.path('/home');
                    } else {
                        $location.path('/confirm');
                    }
                })
                .then(null, function (error) {
                    $dialog.alert(error);
                });
        };

        $scope.init();
    }]);