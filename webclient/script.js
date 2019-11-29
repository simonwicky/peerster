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

	function updatePeers(){

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
					if (list[i].length > 0){
						li.innerHTML = list[i];
						ul.appendChild(li);
					}
				}
				var nodeBox = document.getElementById("node-box");
				nodeBox.innerHTML = "";
				nodeBox.appendChild(ul);
			}
		});
	}

	function updateNodes(){
		$.ajax({
			type: "GET",
			url:"/identifier",
			datatype: "string",
			success: function(data,status){
				var list = data.split(",");
				var select = document.createElement("select");
				select.id = "id-list";
				if (list.length == 1) {
					select.size = 2;
				} else {
					select.size = list.length;
				}
				for(var i = 0; i < list.length; i++){
					var option = document.createElement("option");
					if (list[i].length > 0){
						option.innerHTML = list[i];
						option.value = list[i];
						select.appendChild(option);
					}
				}
				var idBox = document.getElementById("id-box");
				idBox.innerHTML = "";
				idBox.appendChild(select);
			}
		});
	}

	$("#send_mp").click(function(){
		var text = $("#mp").val();
		document.getElementById("mp").value = "";
		var select = document.getElementById("id-list");
		var identifier = select.options[select.selectedIndex].value;
		var mp = {
				Origin:"",
				ID:0,
				Text:text,
				Destination: identifier,
				HopLimit:0,
		};
		$.ajax({
			type: "POST",
			url:"/identifier",
			data:JSON.stringify(mp),
		});
	});


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



	$("#add-peer").click(function(){
		var peer = $("#peer-text").val();
		document.getElementById("peer-text").value = "";

		$.ajax({
			type: "POST",
			url:"/node",
			data:peer,
		});
	});

	$("#share-file").click(function(){
		var name = document.getElementById("file-input").files[0].name;
		$.ajax({
			type: "POST",
			url:"/file",
			data:name,
		});
	});

	$("#dl-file").click(function(){
		var name = $("#dl-name").val();
		document.getElementById("dl-name").value = "";
		var peer = $("#dl-peer").val();
		document.getElementById("dl-peer").value = "";
		var request = $("#dl-request").val();
		document.getElementById("dl-request").value = "";
		var parameters = [name, peer, request]
		$.ajax({
			type: "POST",
			url:"/download",
			data:JSON.stringify(parameters),
		});
	});

	$("#keywords_button").click(function(){
		var keywords = $("#keywords").val();
		var matchBox = document.getElementById("match-box");
		matchBox.innerHTML = "";
		$.ajax({
			type: "POST",
			url:"/keywords",
			data:keywords,
			success:function(data) {
				var list = data.split(",");
				var ul = document.createElement("ul");
				ul.id = "match-list"
				for(var i = 0; i < list.length; i++){
					var li = document.createElement("li");
					li.innerHTML = list[i];
					li.id = "match" + i;
					li.addEventListener("dblclick",function(e){
						download_file(e);
					});
					ul.appendChild(li);
				}
				var matchBox = document.getElementById("match-box");
				matchBox.innerHTML = "";
				matchBox.appendChild(ul);
			}
		});
	});

	function download_file(e){
		$.ajax({
			type: "POST",
			url:"/downloadsearch",
			data: e.target.id[e.target.id.length-1],
		});
	};

	function updateTLCNames(){
		$.ajax({
			type: "GET",
			url:"/TLCNames",
			datatype: "string",
			success: function(data,status){
				var list = data.split(",");
				var ul = document.createElement("ul");
				ul.id = "confirmed-name"
				for(var i = 0; i < list.length; i++){
					var li = document.createElement("li");
					if (list[i].length > 0){
						li.innerHTML = list[i];
						ul.appendChild(li);
					}
				}
				var box = document.getElementById("confirmed-name-box");
				box.innerHTML = "";
				box.appendChild(ul);
			}
		});
	}

	function getID(){
		$.ajax({
			type: "GET",
			url: "/id",
			datatype: "string",
			success: function(data,status) {
				if (data == ""){
					data = "Unknown";
				}
				document.getElementById("my-id").innerHTML = data;
				document.title = "Peerster : " + data;
			}
		})
	}
	function getRound(){
		$.ajax({
			type: "GET",
			url: "/round",
			datatype: "string",
			success: function(data,status) {
				if (data == ""){
					data = "Unknown";
				}
				document.getElementById("my-round").innerHTML = data;
			}
		})
	}

	updatePeers();
	updateMessages();
	getID();
	updateNodes();
	updateTLCNames();
	getRound();
	setInterval(function(){
		updateMessages();
		updatePeers();
		updateNodes();
		updateTLCNames();
		getRound();
	},1000);

});