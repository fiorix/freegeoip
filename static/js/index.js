var gmap="//maps.google.com/maps?f=q&source=s_q&hl=en&geocode=&ie=UTF8&iwloc=A&output=embed&";
function getParameterByName(name) {
	var match = RegExp('[?&]' + name + '=([^&]*)').exec(window.location.search);
	return match && decodeURIComponent(match[1].replace(/\+/g, ' '));
};
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
	$scope.search = function(q, showMap) {
		$scope.error = null;
		$scope.map = showMap;
		$scope.searching = true;
		$http.get('json/' + (q || "")).
		success(function(rs){
			$scope.q = q || rs.ip;
			if(!rs.metro_code) rs.metro_code = '-';
			if(!rs.area_code) rs.area_code = '-';
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
			$("#map").attr("src", gmap+"z="+zoom+"&q="+qs);
			$scope.searching = false;
		}).
		error(function(data,st){
			$scope.errorq = q;
			$scope.error = st;
			$scope.searching = false;
			$("#map").attr("src", gmap+"zoom=2&q=africa");
		});
	}
	$scope.search(getParameterByName("q"), getParameterByName("map"));
}
$(document).ready(function(){
	$("#map").height($("#map").width()/2);
	$("#cb").tooltip();
	$("#cb").click(function(el){el.preventDefault();});
});
