function format_container_names(names) {
  str = '';
  for (var i = 0; i < names.length; i++) {
    str += names[i].substring(1) + "<br />";
  }
  return str;
}

// Borrowed from http://stackoverflow.com/a/6078873
function format_container_time(UNIX_timestamp){
  var a = new Date(UNIX_timestamp*1000);
  var months = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'];
  var year = a.getFullYear();
  var month = months[a.getMonth()];
  var date = a.getDate();
  var hour = a.getHours();
  var min = a.getMinutes();
  var sec = a.getSeconds();
  var time = date + ',' + month + ' ' + year + ' ' + hour + ':' + min + ':' + sec ;
  return time;
}

function load_users() {
  $.get('/api/users', function(data) {
    for (var i = 0; i < data.length; i++) {
      var tr = $('#'+data[i].cid);
      if (!tr) {
        continue
      }
      tr.children('td.user').append(data[i].user_name+'<br />');
    }
  }, 'json');
}

$(document).ready(function() {
  $.get('/api/containers', function(data) {
    var containers = $('#containers');
    for (var i = 0; i < data.length; i++) {
      var tr = $('<tr id="' + data[i].Id + '"></tr>');
      tr.append('<td>' + data[i].Id.substring(0, 12) + '</td>');
      tr.append('<td class="user"></td>');
      tr.append('<td>' + format_container_names(data[i].Names) + '</td>');
      tr.append('<td>' + format_container_time(data[i].Created) + '</td>');
      tr.append('<td>' + data[i].Image + '</td>');
      tr.append('<td>' + data[i].Status + '</td>');
      tr.append('<td><a href="#">Delete</a></td>');
      containers.append(tr);
    }
    load_users();
  }, 'json');
});
