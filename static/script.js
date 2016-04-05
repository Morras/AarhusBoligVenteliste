
function getData() {
    var httpRequest = new XMLHttpRequest();
    httpRequest.onreadystatechange = function() {
        if (httpRequest.readyState == 4) {
            var json = JSON.parse(httpRequest.responseText);
            setNumberOfAppartments(json.NumberOfAppartments);
            setMedian(json.Median);
            createChart(json.Appartments);
        }
    };

    httpRequest.open('GET', 'status');
    httpRequest.send(null);
}

function setNumberOfAppartments(nr){
    document.getElementById('applications').innerHTML = nr;
}

function setMedian(median) {
    document.getElementById("median").innerHTML = median;
}

function createChart(appartments) {

    var dataTable = new google.visualization.DataTable();
    var dataTableColumns = [];

    var first = true;
    appartments.forEach(function(appartment) {
        var tempTable = new google.visualization.DataTable();

        tempTable.addColumn("date", "Date");
        var idSplit = appartment.ID.split("|");
        tempTable.addColumn("number", idSplit[0] + ": " + idSplit[3]);

        var rows = []
        appartment.Ranks.forEach(function(rank) {
            var time = rank.Time.split('T')[0];
            var row = [new Date(time), rank.Rank];
            rows.push(row);
        });
        tempTable.addRows(rows);

        if (first) {
            dataTable = tempTable;
        } else {
            dataTable = google.visualization.data.join(dataTable, tempTable, 'full', [[0, 0]], dataTableColumns, [1])
        }
        dataTableColumns.push(dataTableColumns.length + 1);
        first = false;
    });


    var options = {

        title: 'Top 15 places in queue for Aarhus Bolig',
        legend: { position: 'none', }

    };

    var chart = new google.visualization.LineChart(document.getElementById('chart'));

    chart.draw(dataTable, options);
}