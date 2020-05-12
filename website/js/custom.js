/*************************************************************************
 * Your custom JS file
 *************************************************************************/

(function () {
  "use strict";

  String.prototype.format = function() {
    var str = this;
    for (var i = 0; i < arguments.length; i++) {
      var reg = new RegExp("\\{" + i + "\\}", "gm");
      str = str.replace(reg, arguments[i]);
    }
    return str;
  }
  // The widget object
  var widgets;

  function dateFormatter(dateString) {
      var d = new Date(dateString);
      var day = d.getDate();
      var month = d.getMonth() + 1;
      var year = d.getFullYear();
      if (day < 10) {
          day = "0" + day;
      }
      if (month < 10) {
          month = "0" + month;
      }
      var date = day + "/" + month + "/" + year;

      return date;
  };

  function getNodeList()
  {
      $.ajax({
        url: "https://apiserver.edge-net.org/api/v1/nodes",
        type: "GET",
        contentType: "json",
        beforeSend: function (xhr) {
            /* Authorization header */
            xhr.setRequestHeader("Authorization", "Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6IkxxVFF6WnBEUmNzU2F6UWZWcnRSdlZwUHBxM05VVVhPWUQ1QXAwbEdCRDQifQ.eyJpc3MiOiJrdWJlcm5ldGVzL3NlcnZpY2VhY2NvdW50Iiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9uYW1lc3BhY2UiOiJyZWdpc3RyYXRpb24iLCJrdWJlcm5ldGVzLmlvL3NlcnZpY2VhY2NvdW50L3NlY3JldC5uYW1lIjoicHVibGljLXVzZXItdG9rZW4teHRkNXMiLCJrdWJlcm5ldGVzLmlvL3NlcnZpY2VhY2NvdW50L3NlcnZpY2UtYWNjb3VudC5uYW1lIjoicHVibGljLXVzZXIiLCJrdWJlcm5ldGVzLmlvL3NlcnZpY2VhY2NvdW50L3NlcnZpY2UtYWNjb3VudC51aWQiOiJiMjFiZWZmZi1iMzliLTRhNTQtODdmMS0zZDRiNzNlYjY4YzgiLCJzdWIiOiJzeXN0ZW06c2VydmljZWFjY291bnQ6cmVnaXN0cmF0aW9uOnB1YmxpYy11c2VyIn0.Ldw9PNoD6GeMfWN5i617asSOiqoSp9UXC2oqBWAnAV8_l3WLlb1J4ftVICMhSqngEAD57wj-HP98xeUZ6zucsVk5apWMDuyRYg8X8DfA7X0BRQJpmVXQubFuniEckw23vIWHtoPT8_ryePI8pA9OMAkA3blT69n6VaOh26LtSRu9pmV3hlvwoKTaFfy8QhxzgpRvhmPraqnoREQmACpevomU3K_C12q2gLpABO6ah5LZKb9s4dyc_sJuTIp9PmOdGpN_yjK6hFyd0D2INeYMbNyc9HALRlqSlZS7pdWDFuUVnCcdM96D7SZSjPepcSuNQj16adgJTD3fthJUiBgi6g");
        },
        success: function(result){
          initTable(result)
          initMap(result)
        },
        error: function(error){
          console.log(`Error ${error}`)
        }
      })
  }

  function initTable(result) {
    $(".node-count").html(result.items.length)
    var active = 0;
    for (var i = 0; i < result.items.length; i++) {
      var rowTemplate = "<tr {0}><td class=\"text-center\">{1}</td><td class=\"text-center\">{2}</td><td class=\"text-center\">{3}</td><td class=\"text-center\">{4}</td><td class=\"text-center\">{5}</td><td class=\"text-center\">{6}</td><td class=\"text-center\">{7}</td><td class=\"text-center\">{8}</td></tr>";
      var address = ""
      for (var j = 0; j < result.items[i].status.addresses.length; j++) {
        if (result.items[i].status.addresses[j].type == "InternalIP") {
          address = result.items[i].status.addresses[j].address
        }
      }
      var location =  parseFloat(result.items[i].metadata.labels["edge-net.io/lat"].slice(1, 8)).toFixed(2)+", "+
        parseFloat(result.items[i].metadata.labels["edge-net.io/lon"].slice(1, 8)).toFixed(2);
      var ready = false
      for (var j = 0; j < result.items[i].status.conditions.length; j++) {
        if (result.items[i].status.conditions[j].type == "Ready") {
          ready = result.items[i].status.conditions[j].status
          if (result.items[i].status.conditions[j].status == "True") {
            active++;
          }
        }
      }
      var style = ""
      if (i % 2 === 0) {
        style = "class=\"bg-color-light-gray\""
      }
      $(".node-container").append(rowTemplate.format(style, result.items[i].metadata.name.replace(".edge-net.io", ""), address, location, result.items[i].metadata.labels["edge-net.io/city"].replace("_", " "),
        result.items[i].metadata.labels["edge-net.io/state-iso"], result.items[i].metadata.labels["edge-net.io/country-iso"], dateFormatter(result.items[i].metadata.creationTimestamp),
        ready));
    }
    $(".active-node-count").html(active)
  }

  function initMap(result) {
    var map = new google.maps.Map(
        document.getElementById('map'),
        {center: new google.maps.LatLng(48.0, 2.0), zoom: 16, minZoom: 2});
    var bounds = new google.maps.LatLngBounds();
    var features = []
    for (var i = 0; i < result.items.length; i++) {
      var address = ""
      for (var j = 0; j < result.items[i].status.addresses.length; j++) {
        if (result.items[i].status.addresses[j].type == "InternalIP") {
          address = result.items[i].status.addresses[j].address
        }
      }
      var location =  parseFloat(result.items[i].metadata.labels["edge-net.io/lat"].slice(1, 8)).toFixed(2)+", "+
        parseFloat(result.items[i].metadata.labels["edge-net.io/lon"].slice(1, 8)).toFixed(2);
      var ready = false
      for (var j = 0; j < result.items[i].status.conditions.length; j++) {
        if (result.items[i].status.conditions[j].type == "Ready") {
          ready = result.items[i].status.conditions[j].status
          if (result.items[i].status.conditions[j].status == "True") {
          }
        }
      }
      var contentString =
            '<h1>'+result.items[i].metadata.name+'</h1>'+
            '<ul><li><b>Continent</b> : '+result.items[i].metadata.labels["edge-net.io/continent"].replace("_", " ")+'</li>'+
            '<li><b>Region/State:</b> '+result.items[i].metadata.labels["edge-net.io/state-iso"]+'</li>'+
            '<li><b>Country:</b> '+result.items[i].metadata.labels["edge-net.io/country-iso"]+'</li>'+
            '<li><b>City:</b> '+result.items[i].metadata.labels["edge-net.io/city"].replace("_", " ")+'</li>'+
            '<li><b>Location:</b> '+location+'</li>'+
            '<li><b>IP Address:</b> '+address+'</li>'+
            '<li><b>Ready:</b> '+ready+'</li></ul>';
      features.push({
        position: new google.maps.LatLng(parseFloat(result.items[i].metadata.labels["edge-net.io/lat"].slice(1, 8)), parseFloat(result.items[i].metadata.labels["edge-net.io/lon"].slice(1, 8))),
        title: result.items[i].metadata.name,
        contentString: contentString
      })
    }
    // Create markers.
    for (var i = 0; i < features.length; i++) {
      var marker = new google.maps.Marker({
        position: features[i].position,
        title: features[i].title,
        map: map
      });
      var infowindow = new google.maps.InfoWindow();
      google.maps.event.addListener(marker,'click', (function(marker,content,infowindow){
          return function() {
              infowindow.setContent(content);
              infowindow.open(map,marker);
          };
      })(marker,features[i].contentString,infowindow));

      bounds.extend(marker.position);
    };
    map.initialZoom = true;
    map.fitBounds(bounds);
  }

  function init() {
    // Create the widget object
    widgets = new edaplotjs.Widgets();

    // Create custom tabs
    widgets.createCustomTab({
      selector: "#custom-tab"
    });

    $(document).ready(function(){
       var s = document.createElement("script");
       s.async = true
       s.defer = true
       s.type = "text/javascript";
       s.src  = "https://maps.googleapis.com/maps/api/js?key=AIzaSyBlJ4wNJ-0S1kKMT9x5fZT_20A0qxDyW1k&callback=gmap_draw";
       window.gmap_draw = function(){
         getNodeList()
       };
       $("head").append(s);
    });
  }

  $(init);
})();
