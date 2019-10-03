$(document).ready(function(){

	$("#send").click(function(){

		$.ajax({
			type: "POST",
			url:"/message",
			data:$("#message").val(),
			datatype: "string",
			success:function(data, status){
				alert(data);
			}
		});
	});


});