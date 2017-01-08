/* The MIT License (MIT)

 Copyright (c) <2015>
 - Mathieu Bodjikian <mathieu@bodjikian.fr>
 - Charles-Antoine Mathieu <skatkatt@root.gg>

 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to deal
 in the Software without restriction, including without limitation the rights
 to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:

 The above copyright notice and this permission notice shall be included in
 all copies or substantial portions of the Software.

 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 THE SOFTWARE. */

module.exports = function (grunt) {
    require('load-grunt-tasks')(grunt);

    grunt.initConfig({
        clean: ['public'],
        concat: {
            js_vendors: {
                src: [
                    "bower_components/jquery/dist/jquery.js",
                    "bower_components/bootstrap/dist/js/bootstrap.js",
                    "bower_components/angular/angular.js",
                    "bower_components/ng-file-upload/ng-file-upload-shim.js",
                    "bower_components/ng-file-upload/ng-file-upload.js",
                    "bower_components/angular-sanitize/angular-sanitize.min.js",
                    "bower_components/angular-route/angular-route.js",
                    "bower_components/angular-bootstrap/ui-bootstrap-tpls.js",
                    "bower_components/ng-clip/dest/ng-clip.min.js",
                    "bower_components/angular-markdown-directive/markdown.js",
                    "bower_components/underscore/underscore.js",
                    "bower_components/filesize/lib/filesize.js",
                    "bower_components/zeroclipboard/dist/ZeroClipboard.js",
                    "bower_components/angular-contenteditable/angular-contenteditable.js",
                    "bower_components/showdown/src/showdown.js",
                ],
                dest: 'public/js/vendor.js',

            },
            css_vendors: {
                src: [
                    "bower_components/bootstrap/dist/css/bootstrap.css",
                    "bower_components/bootstrap-flat/css/bootstrap-flat.css",
                    "bower_components/bootstrap-flat/css/bootstrap-flat-extras.css",
                    "bower_components/fontawesome/css/font-awesome.css",
                    "css/water_drop.css"
                ],
                dest: 'public/css/vendor.css'
            }
        },
        copy: {
            stylesheets: {
                files: [{
                    expand: true,
                    src: [
                        'bower_components/bootstrap/fonts/*',
                        'bower_components/fontawesome/fonts/*',
                    ],
                    dest: 'public/fonts/',
                    flatten: true
                }]
            }
        },
        ngAnnotate: {
            options: {
                singleQuotes: true
            },
            all: {
                files: {
                    'public/js/vendor.js': ['public/js/vendor.js']
                }
            }
        },
        uglify: {
            options: {
                mangle: true,
                compress: true,
                report: true,
                sourceMap: true
            },
            javascript: {
                files: {
                    'public/js/vendor.js': ['public/js/vendor.js'],
                }
            }

        },
        cssmin: {
            options: {
                keepSpecialComments: 0
            },
            combine: {
                files: {
                    'public/css/vendor.css': ['public/css/vendor.css']
                }
            }
        }
    });

    grunt.registerTask('default', ['clean', 'concat', 'copy', 'ngAnnotate', 'uglify', 'cssmin']);
};