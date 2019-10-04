$(document).ready(function(){

	function updateMessages(){
		$.ajax({
			type: "GET",
			url:"/message",
			datatype:"application/json",
			success: function(data){
				data = JSON.parse(data);
				var ul = document.createElement("ul");
				ul.id = "message-list"
				for(var i = 0; i < data.length; i++){
					var li = document.createElement("li");
					li.innerHTML = data[i].Origin + "(" + data[i].ID +") : " + data[i].Text;
					ul.appendChild(li);
				}
				var chatBox = document.getElementById("chat-box");
				chatBox.innerHTML = "";
				chatBox.appendChild(ul);
			}
		});
	}

	$("#send").click(function(){
		var text = $("#message").val();
		document.getElementById("message").value = "";
		var rumor = {
				Origin:"",
				ID:0,
				Text:text,
		};
		$.ajax({
			type: "POST",
			url:"/message",
			data:JSON.stringify(rumor),
		});
	});

	$("#refresh-message").click(function(){
		updateMessages()
	});


	$("#add-peer").click(function(){
		var peer = $("#peer-text").val();
		document.getElementById("peer-text").value = "";

		$.ajax({
			type: "POST",
			url:"/node",
			data:peer,
		});
	});

	$("#refresh-peer").click(function(){

		$.ajax({
			type: "GET",
			url:"/node",
			datatype: "string",
			success: function(data,status){
				var list = data.split(",");
				var ul = document.createElement("ul");
				ul.id = "node-list"
				for(var i = 0; i < list.length; i++){
					var li = document.createElement("li");
					li.innerHTML = list[i];
					ul.appendChild(li);
				}
				var nodeBox = document.getElementById("node-box");
				nodeBox.innerHTML = "";
				nodeBox.appendChild(ul);
			}
		});
	});

	
	$.ajax({
		type: "GET",
		url: "/id",
		datatype: "string",
		success: function(data,status) {
			if (data == ""){
				data = "Unknown";
			}
			document.getElementById("my-id").innerHTML = data;
		}
	})

});