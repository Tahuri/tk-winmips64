// RegisterView.cpp : implementation file
//

#include "stdafx.h"
#include "WinEVE.h"
#include "RegisterView.h"
#include "WinEVEDoc.h"

#ifdef _DEBUG
#define new DEBUG_NEW
#undef THIS_FILE
static char THIS_FILE[] = __FILE__;
#endif

/////////////////////////////////////////////////////////////////////////////
// CRegisterView

IMPLEMENT_DYNCREATE(CRegisterView, CScrollView)

CRegisterView::CRegisterView()
{
}

CRegisterView::~CRegisterView()
{
}


BEGIN_MESSAGE_MAP(CRegisterView, CScrollView)
	//{{AFX_MSG_MAP(CRegisterView)
		// NOTE - the ClassWizard will add and remove mapping macros here.
	//}}AFX_MSG_MAP
END_MESSAGE_MAP()

/////////////////////////////////////////////////////////////////////////////
// CRegisterView drawing

void CRegisterView::OnDraw(CDC* pDC)
{
	int i;

	CWinEVEDoc* pDoc = GetDocument();
	char txt[20];

	CFont font;
	font.CreateFont(12,0,0,0,400,FALSE,FALSE,0,
					ANSI_CHARSET,OUT_DEFAULT_PRECIS,CLIP_DEFAULT_PRECIS,
					DEFAULT_QUALITY,DEFAULT_PITCH|FF_MODERN,"Courier");
	pDC->SelectObject(&font);
	
	for (i=0;i<32;i++)
	{
		if (i<10) sprintf(txt,"R%d =  %08x",i,pDoc->r[i]);
		else sprintf(txt,"R%d = %08x",i,pDoc->r[i]);
		pDC->TextOut(0,i*14,txt);
	}

}

/////////////////////////////////////////////////////////////////////////////
// CRegisterView diagnostics

#ifdef _DEBUG
void CRegisterView::AssertValid() const
{
	CScrollView::AssertValid();
}

void CRegisterView::Dump(CDumpContext& dc) const
{
	CScrollView::Dump(dc);
}


CWinEVEDoc* CRegisterView::GetDocument() // non-debug version is inline
{
	ASSERT(m_pDocument->IsKindOf(RUNTIME_CLASS(CWinEVEDoc)));
	return (CWinEVEDoc*)m_pDocument;
}
#endif //_DEBUG

/////////////////////////////////////////////////////////////////////////////
// CRegisterView message handlers

void CRegisterView::OnInitialUpdate() 
{
	CScrollView::OnInitialUpdate();
	
	CSize sizeTotal;
	// TODO: calculate the total size of this view
	sizeTotal.cx = sizeTotal.cy = 100;
	SetScrollSizes(MM_TEXT, sizeTotal);
	// TODO: Add your specialized code here and/or call the base class
	
}
