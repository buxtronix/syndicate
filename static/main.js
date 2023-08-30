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
  $("input[name=untappdid]").val("");
  $("#untappdresult").html("").show();
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
  var comment = button.data('comment')
  var ret = button.data('return')
  var modal = $(this)
  modal.find('.modal-title').text('Checkout ' + bbrewer + ' ' + bname)
  modal.find('.modal-comment').html('<small><i>' + comment + '</i></small>')
  $("input[name=return]").val(ret);
  $("input[name=contid]").val(cid);
})

$('#delCheckoutModal').on('show.bs.modal', function (event) {
  var button = $(event.relatedTarget) // Button that triggered the modal
  var cid = button.data('coid')
  var contid = button.data('contid')
  var modal = $(this)
  $("input[name=coid]").val(cid);
  $("input[name=contid]").val(contid);
})

// Search functionality in /beers
$("#beerSearch").keyup(function() {
    var query = $("#beerSearch").val().toLowerCase();
    $("#BeerTable tr.item").each(function() {
        var thing = $(this).find("button");
        if (thing[0].dataset.beername.toLowerCase().includes(query)) {
            $(this).show();
        } else {
            $(this).hide();
        }
    });
});

$("#beerSearchClear").on('click', function() {
    $("#beerSearch").val("");
    $("#BeerTable tr.item").each(function() {
            $(this).show();
    });
});

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

$('.btn-type').click(function() {
    $(this).addClass('btn-primary').removeClass('btn-default').siblings().removeClass('btn-primary').addClass('btn-default');
});


const applicationServerPublicKey = 'BHVIXApfzS25EkHw0YvpE9rHK31lL57eEyZlFGDlaca7A8LYF9hsqZh8GyB1MBq1CCx8VzeHbjjj6RN9KYo9jSU';

var isSubscribed = false;

if ('serviceWorker' in navigator && 'PushManager' in window) {
      console.log('Service Worker and Push is supported');

      navigator.serviceWorker.register('/static/sw.js')
      .then(function(swReg) {
              console.log('Service Worker is registered', swReg);

              swRegistration = swReg;
              initializeUI();
            })
      .catch(function(error) {
              console.error('Service Worker Error', error);
            });
} else {
      console.warn('Push messaging is not supported');
//      $('#notifyBtn').text('Push Not Supported');
}

function initializeUI() {
//    $('#notifyBtn').click(function(){
//        $('#notifyBtn').attr('disabled', true);
//        if (isSubscribed) {
//            unsubscribeUser();
//        } else {
//            subscribeUser();
//        }
//    });
    swRegistration.pushManager.getSubscription()
    .then(function(subscription) {
        isSubscribed = !(subscription === null);

        if (isSubscribed) {
            console.log('User is subscribed');
        } else {
            console.log('User not subscribed');
            if (Notification.permission != 'denied') {
                subscribeUser();
            }
        }
        updateBtn();
    });
}

function updateBtn() {
    return;
    if (isSubscribed) {
        $('#notifyBtn').text('Disable push');
    } else {
        $('#notifyBtn').text('Enable push');
    }
    $('#notifyBtn').attr('disabled', false);
    if (Notification.permission === 'denied') {
        $('#notifyBtn').text('Push blocked');
        $('#notifyBtn').attr('disabled', true);
        // updateSubscriptionOnServer(null);
        return;
    }

}

/**
 * urlBase64ToUint8Array
 * 
* @param {string} base64String a public vavid key
*/
function urlBase64ToUint8Array(base64String) {
        var padding = '='.repeat((4 - base64String.length % 4) % 4);
        var base64 = (base64String + padding)
            .replace(/\-/g, '+')
            .replace(/_/g, '/');

        var rawData = window.atob(base64);
        var outputArray = new Uint8Array(rawData.length);

        for (var i = 0; i < rawData.length; ++i) {
                    outputArray[i] = rawData.charCodeAt(i);
                }
        return outputArray;
}

function subscribeUser() {
    const applicationServerKey = urlBase64ToUint8Array(applicationServerPublicKey);

    swRegistration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: applicationServerKey
    })
    .then(function(subscription) {
        updateSubscriptionOnServer(subscription, true);
        console.log('User subscribed');
        isSubscribed = true;
        updateBtn();
    })
    .catch(function(err) {
        console.log('Failed to subscribe: ', err);
        updateBtn();
    });
}

function updateSubscriptionOnServer(subscription, enable) {
    console.log(JSON.stringify(subscription));
    var encodedKey = btoa(String.fromCharCode.apply(null, new Uint8Array(subscription.getKey('p256dh'))));
    var encodedAuth = btoa(String.fromCharCode.apply(null, new Uint8Array(subscription.getKey('auth'))));
    var url = enable ? '/subscribe' : '/unsubscribe';
    $.ajax({
        type: 'POST',
        url: url,
        data:{
            endpoint: subscription.endpoint,
            key: encodedKey,
            auth: encodedAuth,
        },
        success: function(response) {
            console.log('Subsribed on server!');
        },
        dataType: 'json'
    }
    );
}

function unsubscribeUser() {
      swRegistration.pushManager.getSubscription()
      .then(function(subscription) {
              if (subscription) {
                        updateSubscriptionOnServer(subscription, false);
                        console.log("Unsub: " + JSON.stringify(subscription));
                        return subscription.unsubscribe();
                      }
            })
      .catch(function(error) {
              console.log('Error unsubscribing', error);
            })
      .then(function() {

              console.log('User is unsubscribed.');
              isSubscribed = false;

              updateBtn();
       });
}
