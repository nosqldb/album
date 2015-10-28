function createEditorMd(divId, submitId, markdown) {
    var editor = editormd(divId, {
        height: 400,
        markdown: markdown,
        autoFocus: false,
        watch : false, 
        path: "http://77fkk5.com1.z0.glb.clouddn.com/static/lib/editor.md-1.5.0/lib/",
        placeholder: "Markdown,提交前请预览格式",
        toolbarIcons: function() {
          return ["undo", "redo", "|", "bold", "italic", "quote", "|", "h1", "h2", "h3", "|", "list-ul", "list-ol", "hr", "|", "link", "reference-link", "image", "code", "code-block", "table", "|", "watch", "preview", "fullscreen", "|", "help", "info"]
        },
        saveHTMLToTextarea: true,
        imageUpload: true,
        imageFormats: ["jpg", "jpeg", "gif", "png"],
        imageUploadURL: "/upload/image",
        onchange: function() {
          $(submitId).attr('disabled', this.getMarkdown().trim() == "");
        }
      });

    return editor;
}

$(document).ready(function(){
    editormd.urls.atLinkBase = "/user/";

    $("[data-toggle=popover]").popover();

    setToTop();

    $('.editormd-preview-container pre').addClass("prettyprint linenums");
    prettyPrint();

    $('.content .body a').attr('target', '_blank');
    $('.reply-content a ').attr('target', '_blank');
});
