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

function dispatch() {
  var href = location.href;
  var routes = {
    '/containers': {
      title: 'Containers',
      action: load_containers,
      nav: 'nav-containers'
    },
    '/users': {
      title: 'Users',
      action: list_users,
      nav: 'nav-users'
    }
  };
  var rt = routes[location.pathname];
  if (!rt) {
    $('#body').html('Page not found. <a href="javascript:location.history.go(-1);">Back to previous page.</a>');
    return;
  }
  $('#page-header').html(rt.title);
  if (rt.action) {
    rt.action();
  }
  if (rt.nav) {
    $('#sidebar-nav').children('li').removeClass('active');
    $('#'+rt.nav).addClass('active');
  }
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

function set_table_header(th) {
  var container = $('#main-table').find('thead');
  var tr = $('<tr></tr>');
  for (var i = 0; i < th.length; i++) {
    tr.append('<th>' + th[i] + '</th>');
  }
  container.append(tr);
}

function load_containers() {
  var th = ["ID", "Users", "Name", "Created At", "Image", "Status", "Action"];
  set_table_header(th);
  $.get('/api/containers', function(data) {
    var container = $('#main-table').find('tbody');
    for (var i = 0; i < data.length; i++) {
      var tr = $('<tr id="' + data[i].Id + '"></tr>');
      tr.append('<td>' + data[i].Id.substring(0, 12) + '</td>');
      tr.append('<td class="user"></td>');
      tr.append('<td>' + format_container_names(data[i].Names) + '</td>');
      tr.append('<td>' + format_container_time(data[i].Created) + '</td>');
      tr.append('<td>' + data[i].Image + '</td>');
      tr.append('<td>' + data[i].Status + '</td>');
      tr.append('<td><a href="#">Grant</a></td>');
      container.append(tr);
    }
    load_users();
  }, 'json');
}

function list_users() {
  var th = ["Name", "Containers", "Created At", "Action"];
  set_table_header(th);
  $.get('/api/users', function(data) {
    var users = {};
    for (var i = 0; i < data.length; i++) {
      var user = data[i];
      if (users[user.user_name]) {
        users[user.user_name].containers.push({cid: user.cid, priv_id: user.priv_id});
      } else {
        users[user.user_name] = {
          id: user.user_id,
          name: user.user_name,
          created_at: user.user_created_at,
          containers: [{cid: user.cid, priv_id: user.priv_id}]
        };
      }
    }
    var container = $('#main-table').find('tbody');
    for (var name in users) {
      var user = users[name];
      var tr = $('<tr id="user_' + user.id + '"></tr>');
      tr.append('<td>' + name + '</td>');
      var td = $('<td></td>');
      for (var i = 0; i < user.containers.length; i++) {
        if (user.containers[i].cid == '') {
          continue;
        }
        td.append('<span id="priv_' + user.containers[i].priv_id + '">' + user.containers[i].cid.substring(0, 12) + '&nbsp;&nbsp;&nbsp;&nbsp;<a href="javascript:revoke(' + user.containers[i].priv_id + ');">Revoke</a></span><br />');
      }
      tr.append(td);
      tr.append('<td>' + user.created_at + '</td>');
      tr.append('<td><a href="javascript:delete_user(' + user.id + ')">Delete</a></td>');
      container.append(tr);
    }
  }, 'json');
}

function revoke(id) {
  $.ajax({
    type: 'DELETE',
    url: '/api/priv/'+id,
    success: function(r) {
      $('#priv_'+id).remove();
    }
  });
}

function delete_user(id) {
  $.ajax({
    type: 'DELETE',
    url: '/api/users/'+id,
    success: function(r) {
      $('#user_'+id).remove();
    }
  });
}

$(document).ready(function() {
  dispatch();
});
