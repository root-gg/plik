// Editable file name directive
angular.module('contentEditable', []).directive('contenteditable', [function () {
    return {
        restrict: 'A',          // only activate on element attribute
        require: '?ngModel',    // get a hold of NgModelController
        scope: {
            invalidClass: '@',  // Bind invalid-class attr evaluated expr
            validator: '&'      // Bind parent scope value
        },
        link: function (scope, element, attrs, ngModel) {
            if (!ngModel) return; // do nothing if no ng-model
            scope.validator = scope.validator(); // ???

            // Update view from model
            ngModel.$render = function () {
                var string = ngModel.$viewValue;
                validate(string);
                element.text(string);
            };

            // Update model from view
            function update() {
                var string = element.text();
                validate(string);
                ngModel.$setViewValue(string);
            }

            // Validate input and update css class
            function validate(string) {
                if (scope.validator) {
                    if (scope.validator(string)) {
                        element.removeClass(scope.invalidClass);
                    } else {
                        element.addClass(scope.invalidClass);
                    }
                }
            }

            // Listen for change events to enable binding
            element.on('blur keyup change', function () {
                scope.$evalAsync(update);
            });
        }
    };
}]);