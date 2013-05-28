angular.module('freegeoip',[])
  .directive('btnSubmit', function(){
    return function(scope,element,attrs){
      scope.$watch(function(){
        return scope.$eval(attrs.btnSubmit);
      },
      function(loading){
        if(loading) $(element).button('loading');
        else $(element).button('reset');
      });
    }
  });
freegeoip.$inject = ['$scope','$http'];
function freegeoip($scope,$http){
  $scope.geoip = {}
  $scope.search = function(q) {
    $scope.error = null;
    $scope.searching = true;
    $http.get('json/'+q).
      success(function(rs){
        //$scope.q = rs.ip;
        if(!rs.metrocode) rs.metrocode = '-';
        if(!rs.areacode) rs.areacode = '-';
        $scope.geoip = angular.copy(rs);
        var qs = "";
        var zoom = 2;
        if(rs.country_name&&rs.country_code!='RD'){
          if(rs.region_name){
            if(rs.city){
              zoom=6;
              qs=rs.city+','+rs.region_name+','+rs.country_name;
            } else {
              zoom=4;
              qs=rs.region_name+','+rs.country_name;
            }
          } else {
              qs=rs.country_name;
          }
        } else {
          qs = "Africa";
        }
        $("#map").attr("src", "http://maps.google.com/maps?f=q&source=s_q&hl=en&geocode=&ie=UTF8&iwloc=A&output=embed&"+"z="+zoom+"&q="+qs);
        $scope.searching = false;
      }).
      error(function(data,status){
        $scope.errorq = q;
        $scope.error = status;
        $scope.searching = false;
        $("#map").attr("src", "http://maps.google.com/maps?f=q&source=s_q&hl=en&geocode=&ie=UTF8&iwloc=A&output=embed&zoom=2&q=africa");
      });
  }
  $scope.search('');
}
$(document).ready(function(){
  $("#map").height($("#map").width()/2);
  $("#cb").tooltip();
  $("#cb").click(function(el){el.preventDefault();});
});
