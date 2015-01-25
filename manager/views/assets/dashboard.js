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
      nav: 'nav-containers',
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
    normalize_user_list(data);
    for (var i = 0; i < data.length; i++) {
      var tr = $('#'+data[i].cid);
      if (!tr) {
        continue
      }
      var user = data[i];
      tr.children('td.user').append('<span id="priv_' + user.priv_id + '">' + user.user_name + '&nbsp;&nbsp;&nbsp;&nbsp;<a href="javascript:revoke(' + user.priv_id + ');">Revoke</a></span><br />');
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
  $('#page-header').append('<button class="btn btn-success pull-right" type="button" data-toggle="modal" data-target="#create_container_box"><span class="glyphicon glyphicon-plus" aria-hidden="true"></span> Create Container</button>');
  $.get('/api/containers', function(data) {
    window.containers = {};
    var container = $('#main-table').find('tbody');
    for (var i = 0; i < data.length; i++) {
      window.containers[data[i].Id] = data[i];
      var tr = $('<tr id="' + data[i].Id + '"></tr>');
      tr.append('<td>' + data[i].Id.substring(0, 12) + '</td>');
      tr.append('<td class="user"></td>');
      tr.append('<td>' + format_container_names(data[i].Names) + '</td>');
      tr.append('<td>' + format_container_time(data[i].Created) + '</td>');
      tr.append('<td>' + data[i].Image + '</td>');
      tr.append('<td>' + data[i].Status + '</td>');
      tr.append('<td><a href="javascript:show_grant_box(\'' + data[i].Id +'\');">Grant</a> | <a href="javascript:delete_container(\''+data[i].Id+'\');"><span class="label label-danger">Delete</span></a></td>');
      container.append(tr);
    }
    load_users();
  }, 'json');
}

function normalize_user_list(data) {
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
  window.users = users;
  return users;
}

function list_users() {
  var th = ["Name", "Containers", "Created At", "Action"];
  set_table_header(th);
  $('#page-header').append('<button class="btn btn-success pull-right" type="button" data-toggle="modal" data-target="#create_user_box"><span class="glyphicon glyphicon-plus" aria-hidden="true"></span> Create User</button>');
  $.get('/api/users', function(data) {
    var users = normalize_user_list(data);
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
    },
    error: error_callback
  });
}

function delete_user(id) {
  $.ajax({
    type: 'DELETE',
    url: '/api/users/'+id,
    success: function(r) {
      $('#user_'+id).remove();
    },
    error: error_callback
  });
}

function create_user() {
  var name = $('#user_name').val();
  var pass = $('#user_password').val();
  if (! /^[a-z0-9]+$/.test(name) || name.length < 3) {
    alert('Username can only contain 0-9a-z and more than 3 characters.');
    return;
  }
  if (pass.length < 6) {
    alert("Password must be more than 6 characters.");
    return;
  }
  $.ajax({
    type: 'POST',
    url: '/api/users',
    data: JSON.stringify({name: name, password: pass}),
    success: function() {
      window.location.reload();
    },
    contentType: 'application/json',
    error: error_callback
  });
}

function create_container() {
  var name = $('#container_name_input').val();
  var image = $('#image_input').val();
  if (! /^[a-z0-9-_]+$/.test(name) || name.length < 3) {
    alert('Username can only contain 0-9a-z-_ and more than 2 characters.');
    return;
  }
  if (image.length < 3) {
    alert('Please select an image');
    return;
  }
  var btn = $('#btn-create-container').button('loading');
  $.ajax({
    type: 'POST',
    url: '/api/containers',
    data: JSON.stringify({name: name, image: image}),
    success: function() {
      window.location.reload();
    },
    contentType: 'application/json',
    error: error_callback
  });
}

function chpasswd() {
  var pass = $('#new_password').val();
  if (pass.length < 6) {
    alert("Password must be more than 6 characters.");
    return;
  }
  $.ajax({
    type: 'POST',
    url: '/api/passwd',
    data: JSON.stringify({password: pass}),
    success: function() {
      alert('Your password has be modifed.');
      $('#chpasswd_box').modal('hide');
    },
    contentType: 'application/json',
    error: error_callback
  });
}

function show_help() {
}

function show_grant_box(id) {
  $('#grant_box').modal('show');
  var names = window.containers[id].Names;
  if (names.length == 0 ) {
    names.push(window.containers[id].Id.substring(0, 12));
  } else {
    names[0] = names[0].substring(1);
  }
  var html = 'Grant access privilege to user for ' + '<span class="label label-info">' + names[0] + '</span>';
  $('#grant_box').find('.modal-title').html(html);
  $('#grant_btn').attr('cid', id);
  $('#select_user_name').empty();
  for (var name in window.users) {
    $('#select_user_name').append('<option value="' + window.users[name].id + '">' + name + '</option>');
  }
}

function grant() {
  var cid = $('#grant_btn').attr('cid');
  var uid = $('#select_user_name').val();
  var user;
  for (var i in window.users) {
    if (window.users[i].id + '' == uid + '') {
      user = window.users[i];
    }
  }
  $.post('/api/priv', {user_id: uid, container: cid}, function(r) {
    $('#'+cid).children('td.user').append('<span id="priv_' + r.id + '">' + user.name + '&nbsp;&nbsp;&nbsp;&nbsp;<a href="javascript:revoke(' + r.id + ');">Revoke</a></span><br />');
    $('#grant_box').modal('hide');
  }, 'json')
}

function logout() {
  $.ajax({
    type: 'DELETE',
    url: '/api/logout',
    success: function(r) {
      location.href = '/';
    }
  });
}

function delete_container(cid) {
  if (!window.confirm('Are you sure to delete container '+cid)) {
    return;
  }

  $.ajax({
    type: 'DELETE',
    url: '/api/containers/'+cid,
    success: function(r) {
      window.location.reload();
    },
    error: error_callback
  })
}

function error_callback(xhr) {
  if (xhr.responseText.length > 0) {
    var data = JSON.parse(xhr.responseText);
    if (data.message) {
      alert(data.message);
    }
  }
}

function bind_login_event() {
  $("#signinForm").submit(function(e) {
    e.preventDefault();
    var data = {
      name: $('#inputUsername').val(),
      password: $('#inputPassword').val()
    };
    $.ajax({
      type: 'POST',
      url: '/api/login', 
      data: JSON.stringify(data),
      contentType: 'application/json',
      success: function(r) {
        location.href = '/containers';
      },
      error: error_callback
    });
  });
}

$(document).ready(function() {
  dispatch();
});
