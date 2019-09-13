$('#addModal').on('show.bs.modal', function (event) {
  var button = $(event.relatedTarget) // Button that triggered the modal
  var bid = button.data('beerid')
  var bname = button.data('beername')
  var bbrewer = button.data('brewer')
  // If necessary, you could initiate an AJAX request here (and then do the updating in a callback).
  // Update the modal's content. We'll use jQuery here, but you could use a data binding library or other methods instead.
  var modal = $(this)
  modal.find('.modal-title').text('Contribute ' + bbrewer + ' ' + bname)
  $("input[name=beerid]").val(bid);
})

$('#addUserModal').on('show.bs.modal', function (event) {
  setTimeout(function() {
    $("input[name=username]").focus();
  }, 1000);
})

$('#addBeerModal').on('show.bs.modal', function (event) {
  var button = $(event.relatedTarget) // Button that triggered the modal
  var modal = $(this)
  $("input[name=untappdid]").val();
  setTimeout(function() {
    $("input[name=untappdid]").focus();
  }, 1000);
})

$('#addContModal').on('show.bs.modal', function (event) {
  var button = $(event.relatedTarget) // Button that triggered the modal
  var bid = button.data('beerid')
  var bname = button.data('beername')
  var bbrewer = button.data('brewer')
  var modal = $(this)
  modal.find('.modal-title').text('Contribute ' + bbrewer + ' ' + bname)
  $("input[name=beerid]").val(bid);
  $("input[name=quantity]").val("");
  $("input[name=unitprice]").val("");
  $("input[name=totalprice]").val("");
})

$('#takeContModal').on('show.bs.modal', function (event) {
  var button = $(event.relatedTarget) // Button that triggered the modal
  var cid = button.data('contid')
  var bname = button.data('beername')
  var bbrewer = button.data('brewer')
  var modal = $(this)
  modal.find('.modal-title').text('Checkout ' + bbrewer + ' ' + bname)
  $("input[name=contid]").val(cid);
  $("input[name=quantity]").val("");
})

$('#delCheckoutModal').on('show.bs.modal', function (event) {
  var button = $(event.relatedTarget) // Button that triggered the modal
  var cid = button.data('coid')
  var contid = button.data('contid')
  var modal = $(this)
  $("input[name=coid]").val(cid);
  $("input[name=contid]").val(contid);
})

var untappdtimer;
$("#untappdid").keyup(function() {
  if (untappdtimer) {
	  clearTimeout(untappdtimer);
  }
  untappdtimer = setTimeout(function() {
	  var value = $("#untappdid").val();
	  if (value == "") {
	    $("#untappdresult").html("");
	  } else {
	    $.post({
	      type: "POST",
		    url: "/untappd/beer",
		    data: {
			    id: value
		    },
		    success: function(html) {
			    $("#untappdresult").html(html).show();
		    }
	      });
	  }
  }, 1000);
  });
