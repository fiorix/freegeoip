function ClientGetLocation(){
	try{
		var ClientGeo = google.gears.factory.create('beta.geolocation');
		ClientGeo.getCurrentPosition(successCallback, errorCallback);
	}
	catch(e){
		try{
			navigator.geolocation.getCurrentPosition(successCallback, errorCallback);
		}
		catch(e){
			errorCallback({code:2,message:e.message});
		}
	}
}

function errorCallback(err){}

function successCallback(r){
	$.ajax({
		async: true,
		type: "GET",
		url: '/submit',
		data: { 
			latitude: r.coords.latitude,
			longitude: r.coords.longitude
		},
		success: function(msg){},
		error: function(err){ 
			alert('Error:' + err.responseText + '  Status: ' + err.status);
		}
	});
}
