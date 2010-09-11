// JavaScript Document
var mapurl = "http://maps.google.com/maps?f=q&source=s_q&hl=en&geocode=&ie=UTF8&iwloc=A&output=embed&key=ABQIAAAAXO_Kw_lKht5dqI_aquiQoBQS8iIBtb3anpUmgvwbGZJdCW94LRTNcVhbmU3bFvhyw9G2yabBQRGD8w&";

function showResult(rs) {
    var zoom = 0;
    var location = "";

    if(rs.country_code == "RD") { return false; }

    if(rs.country_name && rs.region_name && rs.city) {
        zoom = 9;
        location = rs.city + ", " + rs.region_name + ", " + rs.country_name

    } else if(rs.country_name && rs.region_name) {
        zoom = 4;
        location = rs.region_name + ", " + rs.country_name;

    } else {
        zoom = 3;
        if(rs.latitude && rs.longitude) {
            location = rs.latitude.toFixed(2) + " " + rs.longitude.toFixed(2);

        } else if(rs.country_name) { 
            location = rs.countryname;

        } else {
            return false;
        }
    }

    $("#map").attr("src", mapurl+"z="+zoom+"&q="+location);
    return true;
}

$(document).ready(function(){
	// widgets and dialogs
	$("#left").corner("10px tl bl");
	$("#error").dialog({autoOpen:false,width:600,modal:true,show:'puff',hide:'puff',buttons:{"Ok": function() {$(this).dialog("close");}}, title:'Ooops!'});
	
	// buttons
	$(".menuitem").click(function(){ $(this).effect("pulsate", {times:1}, 1000); });
	$("#apidocs_button").click(function(){ location.href="http://github.com/fiorix/freegeoip/blob/master/README.rst"; });
	$("#download_button").click(function(){ location.href="http://github.com/fiorix/freegeoip"; });

	// maps
	$("#geoip").submit(function(){
		var addr = $("input:first").val();
		if(addr && addr != "ip or hostname") {
			$("#searchbutton").effect("pulsate", {times:1}, 1000);
			$.getJSON("/json/"+addr, function(rs){ if(!showResult(rs)) { $("#error").dialog("open"); } });
		}
	});
	
	$.getJSON("/json/", function(rs) {
		if(rs.ip) { $("#inputbox").attr("value", rs.ip); }
		if(!showResult(rs)) { $("#map").attr("src", mapurl+"z=0&q=africa"); }
	});
	
	return false;
});
