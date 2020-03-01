// Modal dialog service
angular.module('dialog', ['ui.bootstrap']).factory('$dialog', function ($uibModal) {

    var module = {};

    // Define error partial here so we can display a connection error
    // without having to load the template from the server
    var alertTemplate = '<div class="modal-header">' + "\n";
    alertTemplate += '<h1>{{title}}</h1>' + "\n";
    alertTemplate += '</div>' + "\n";
    alertTemplate += '<div class="modal-body">' + "\n";
    alertTemplate += '<p>{{message}}</p>' + "\n";
    alertTemplate += '<p ng-show="data.value">' + "\n";
    alertTemplate += '{{value}}' + "\n";
    alertTemplate += '</p>' + "\n";
    alertTemplate += '</div>' + "\n";
    alertTemplate += '<div class="modal-footer" ng-if="confirm">' + "\n";
    alertTemplate += '<button ng-click="$dismiss()" class="btn btn-danger">Cancel</button>' + "\n";
    alertTemplate += '<button ng-click="$close()" class="btn btn-success">OK</button>' + "\n";
    alertTemplate += '</div>' + "\n";
    alertTemplate += '<div class="modal-footer" ng-if="!confirm">' + "\n";
    alertTemplate += '<button ng-click="$close()" class="btn btn-primary">Close</button>' + "\n";
    alertTemplate += '</div>' + "\n";

    // alert dialog
    module.alert = function (data) {
        var options = {
            backdrop: true,
            backdropClick: true,
            template: alertTemplate,
            controller: 'AlertDialogController',
            resolve: {
                args: function () {
                    return {
                        data: angular.copy(data)
                    };
                }
            }
        };

        return module.openDialog(options);
    };

    // generic dialog
    module.openDialog = function (options) {
        return $uibModal.open(options);
    };

    return module;
});

// Alert modal dialog controller
plik.controller('AlertDialogController', ['$scope', 'args',
    function ($scope, args) {

        _.extend($scope, args.data);

        if (!$scope.title) {
            if ($scope.status) {
                if ($scope.status === 100) {
                    $scope.title = 'Success !';
                } else {
                    $scope.title = 'Oops ! (' + $scope.status + ')';
                }
            }
        }
    }]);

// HTTP basic auth credentials dialog controller
plik.controller('PasswordController', ['$scope',
    function ($scope) {

        // Ugly but it works
        setTimeout(function () {
            $("#login").focus();
        }, 100);

        $scope.title = 'Please fill credentials !';
        $scope.login = 'plik';
        $scope.password = '';

        $scope.close = function (login, password) {
            if (login.length > 0 && password.length > 0) {
                $scope.$close({login: login, password: password});
            }
        };
    }]);

// QRCode dialog controller
plik.controller('QRCodeController', ['$scope', 'args',
    function ($scope, args) {
        $scope.args = args;
    }]);