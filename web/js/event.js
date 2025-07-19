$(document).ready(function () {
  // ✅ 基础表格初始化
  $("#basic-datatables").DataTable();

  // ✅ 多列筛选表格初始化
  $("#multi-filter-select").DataTable({
    pageLength: 5,
    initComplete: function () {
      this.api()
        .columns()
        .every(function () {
          const column = this;
          const select = $(
            '<select class="form-control"><option value=""></option></select>'
          )
            .appendTo($(column.footer()).empty())
            .on("change", function () {
              const val = $.fn.dataTable.util.escapeRegex($(this).val());
              column.search(val ? "^" + val + "$" : "", true, false).draw();
            });

          column
            .data()
            .unique()
            .sort()
            .each(function (d) {
              select.append('<option value="' + d + '">' + d + "</option>");
            });
        });
    },
  });

  // ✅ Add Row 表格初始化
  $("#add-row").DataTable({
    pageLength: 5,
  });

  // ✅ 添加新行按钮逻辑
  const actionBtns = `
    <td>
      <div class="form-button-action">
        <button type="button" data-toggle="tooltip" title="Edit Task" class="btn btn-link btn-primary btn-lg">
          <i class="fa fa-edit"></i>
        </button>
        <button type="button" data-toggle="tooltip" title="Remove" class="btn btn-link btn-danger">
          <i class="fa fa-times"></i>
        </button>
      </div>
    </td>`;

  $("#addRowButton").click(function () {
    $("#add-row")
      .dataTable()
      .fnAddData([
        $("#addName").val(),
        $("#addPosition").val(),
        $("#addOffice").val(),
        actionBtns,
      ]);
    $("#addRowModal").modal("hide");
  });
});
